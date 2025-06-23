package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	_ "github.com/glebarez/go-sqlite"
	"github.com/go-redis/redis/v8"
	"github.com/joho/godotenv"
)

var db *sql.DB
var rdb *redis.Client
var ctx = context.Background()

type AIConfig struct {
	APIKey  string
	BaseURL string
	Model   string
}

var aiConfig AIConfig

// Session management
type Session struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	TableName string    `json:"table_name"`
	CreatedAt time.Time `json:"created_at"`
	LastUsed  time.Time `json:"last_used"`
}

// Request structs with session support
type QueryRequest struct {
	SQL       string `json:"sql"`
	SessionID string `json:"session_id"`
}

type NaturalRequest struct {
	Query     string `json:"query"`
	SessionID string `json:"session_id"`
}

type UploadRequest struct {
	SessionID string `json:"session_id,omitempty"`
}

// Response structs
type QueryResponse struct {
	Data      []map[string]interface{} `json:"data"`
	Runtime   string                   `json:"runtime"`
	Query     string                   `json:"query"`
	RowCount  int                      `json:"row_count"`
	SessionID string                   `json:"session_id"`
	Cached    bool                     `json:"cached"`
}

type NaturalResponse struct {
	NaturalQuery string                   `json:"natural_query"`
	GeneratedSQL string                   `json:"generated_sql"`
	Explanation  string                   `json:"explanation"`
	Runtime      string                   `json:"runtime"`
	Data         []map[string]interface{} `json:"data"`
	RowCount     int                      `json:"row_count"`
	SessionID    string                   `json:"session_id"`
	Cached       bool                     `json:"cached"`
}

type UploadResponse struct {
	Message   string   `json:"message"`
	Filename  string   `json:"filename"`
	Rows      int      `json:"rows"`
	Columns   []string `json:"columns"`
	Table     string   `json:"table"`
	SessionID string   `json:"session_id"`
}

type SessionResponse struct {
	SessionID string `json:"session_id"`
	Message   string `json:"message"`
}

// AI request/response structs
type AIRequest struct {
	Model     string    `json:"model"`
	MaxTokens int       `json:"max_tokens"`
	Messages  []Message `json:"messages"`
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type AIResponse struct {
	Content []struct {
		Text string `json:"text"`
	} `json:"content"`
}

type SQLResponse struct {
	SQL string `json:"sql"`
}

func init() {
	// Load .env file
	if err := godotenv.Load(); err != nil {
		log.Println("Warning: No .env file found")
	}

	aiConfig = AIConfig{
		APIKey:  os.Getenv("AI_API_KEY"),
		BaseURL: "https://api.anthropic.com",
		Model:   "claude-3-5-sonnet-20241022",
	}

	// Initialize Redis
	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "localhost:6379"
	}

	redisPassword := os.Getenv("REDIS_PASSWORD")
	redisDB := 0

	rdb = redis.NewClient(&redis.Options{
		Addr:         redisAddr,
		Password:     redisPassword,
		DB:           redisDB,
		DialTimeout:  10 * time.Second,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		PoolSize:     10,
		PoolTimeout:  30 * time.Second,
	})

	// Test Redis connection
	if _, err := rdb.Ping(ctx).Result(); err != nil {
		log.Printf("Warning: Redis connection failed: %v", err)
	} else {
		log.Println("Redis connected successfully")
	}

	if aiConfig.APIKey == "" {
		log.Println("Warning: AI_API_KEY not found in environment")
	} else {
		log.Println("AI configuration loaded successfully")
	}
}

func main() {
	var err error
	db, err = sql.Open("sqlite", ":memory:")
	if err != nil {
		panic(err)
	}
	defer db.Close()

	router := gin.Default()
	router.Use(cors.Default())

	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "OK",
			"redis":  checkRedisHealth(),
		})
	})

	// Session management
	router.POST("/api/session/create", handleCreateSession)
	router.DELETE("/api/session/:sessionId", handleDeleteSession)
	router.GET("/api/session/:sessionId/status", handleSessionStatus)

	// Data operations with session support
	router.POST("/api/upload", handleUpload)
	router.POST("/api/query", handleQuery)
	router.POST("/api/natural", handleNatural)
	router.GET("/api/schema/:sessionId", handleSchema)

	// Start cleanup routine for expired sessions
	go cleanupExpiredSessions()

	router.Run(":8080")
}

// Session management functions
func generateSessionID() string {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return fmt.Sprintf("session_%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(bytes)
}

func handleCreateSession(c *gin.Context) {
	sessionID := generateSessionID()
	tableName := fmt.Sprintf("data_%s", sessionID)

	session := Session{
		ID:        sessionID,
		TableName: tableName,
		CreatedAt: time.Now(),
		LastUsed:  time.Now(),
	}

	// Store session in Redis
	sessionData, _ := json.Marshal(session)
	if err := rdb.Set(ctx, fmt.Sprintf("session:%s", sessionID), sessionData, 24*time.Hour).Err(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create session"})
		return
	}

	c.JSON(http.StatusOK, SessionResponse{
		SessionID: sessionID,
		Message:   "Session created successfully",
	})
}

func handleDeleteSession(c *gin.Context) {
	sessionID := c.Param("sessionId")

	// Delete session data from Redis
	pipe := rdb.Pipeline()
	pipe.Del(ctx, fmt.Sprintf("session:%s", sessionID))
	pipe.Del(ctx, fmt.Sprintf("cache:query:%s:*", sessionID))
	pipe.Del(ctx, fmt.Sprintf("cache:natural:%s:*", sessionID))
	pipe.Del(ctx, fmt.Sprintf("schema:%s", sessionID))

	if _, err := pipe.Exec(ctx); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete session"})
		return
	}

	// Drop table if it exists
	tableName := fmt.Sprintf("data_%s", sessionID)
	db.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s", tableName))

	c.JSON(http.StatusOK, gin.H{"message": "Session deleted successfully"})
}

func handleSessionStatus(c *gin.Context) {
	sessionID := c.Param("sessionId")

	sessionData, err := rdb.Get(ctx, fmt.Sprintf("session:%s", sessionID)).Result()
	if err == redis.Nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Session not found"})
		return
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get session"})
		return
	}

	var session Session
	json.Unmarshal([]byte(sessionData), &session)

	c.JSON(http.StatusOK, session)
}

func getSession(sessionID string) (*Session, error) {
	if sessionID == "" {
		return nil, fmt.Errorf("session ID is required")
	}

	sessionData, err := rdb.Get(ctx, fmt.Sprintf("session:%s", sessionID)).Result()
	if err == redis.Nil {
		return nil, fmt.Errorf("session not found")
	} else if err != nil {
		return nil, err
	}

	var session Session
	if err := json.Unmarshal([]byte(sessionData), &session); err != nil {
		return nil, err
	}

	// Update last used time
	session.LastUsed = time.Now()
	updatedData, _ := json.Marshal(session)
	rdb.Set(ctx, fmt.Sprintf("session:%s", sessionID), updatedData, 24*time.Hour)

	return &session, nil
}

func handleUpload(c *gin.Context) {
	sessionID := c.PostForm("session_id")
	log.Printf("Received session_id: %s", sessionID)

	session, err := getSession(sessionID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Invalid session: %v", err)})
		return
	}

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to get file"})
		return
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read CSV file"})
		return
	}

	if len(records) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "CSV file is empty"})
		return
	}

	headers := records[0]
	tableName := session.TableName

	// Drop existing table and clear related caches
	db.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s", tableName))
	clearSessionCaches(session.ID)

	// Create table with quoted column names
	columns := make([]string, len(headers))
	for i, header := range headers {
		cleanHeader := strings.TrimSpace(header)
		columns[i] = fmt.Sprintf("`%s` TEXT", cleanHeader)
	}

	createSQL := fmt.Sprintf("CREATE TABLE %s (%s)", tableName, strings.Join(columns, ", "))

	if _, err = db.Exec(createSQL); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Failed to create table: %v", err),
			"sql":   createSQL,
		})
		return
	}

	// Insert data
	for _, record := range records[1:] {
		if len(record) != len(headers) {
			continue
		}

		placeholders := strings.Repeat("?,", len(record)-1) + "?"
		insertSQL := fmt.Sprintf("INSERT INTO %s VALUES (%s)", tableName, placeholders)

		values := make([]interface{}, len(record))
		for j, v := range record {
			values[j] = strings.TrimSpace(v)
		}

		if _, err = db.Exec(insertSQL, values...); err != nil {
			continue
		}
	}

	// Cache the schema
	schema, _ := getTableSchemaForSession(session.ID, tableName)
	rdb.Set(ctx, fmt.Sprintf("schema:%s", session.ID), schema, 24*time.Hour)

	c.JSON(http.StatusOK, UploadResponse{
		Message:   "File uploaded successfully",
		Filename:  header.Filename,
		Rows:      len(records) - 1,
		Columns:   headers,
		Table:     tableName,
		SessionID: session.ID,
	})
}

func handleQuery(c *gin.Context) {
	start := time.Now()
	var request QueryRequest

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON"})
		return
	}

	session, err := getSession(request.SessionID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Invalid session: %v", err)})
		return
	}

	// Check cache first
	cacheKey := fmt.Sprintf("cache:query:%s:%s", session.ID, hashString(request.SQL))
	if cachedResult, err := rdb.Get(ctx, cacheKey).Result(); err == nil {
		var cachedResponse QueryResponse
		if json.Unmarshal([]byte(cachedResult), &cachedResponse) == nil {
			cachedResponse.Cached = true
			c.JSON(http.StatusOK, cachedResponse)
			return
		}
	}

	// Execute query
	rows, err := db.Query(request.SQL)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to execute query: %v", err)})
		return
	}
	defer rows.Close()

	results, _ := processRows(rows)
	duration := time.Since(start)

	response := QueryResponse{
		Data:      results,
		Runtime:   duration.String(),
		Query:     request.SQL,
		RowCount:  len(results),
		SessionID: session.ID,
		Cached:    false,
	}

	// Cache the result for 1 hour
	if responseData, err := json.Marshal(response); err == nil {
		rdb.Set(ctx, cacheKey, responseData, time.Hour)
	}

	c.JSON(http.StatusOK, response)
}

func handleNatural(c *gin.Context) {
	var request NaturalRequest

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON"})
		return
	}

	session, err := getSession(request.SessionID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Invalid session: %v", err)})
		return
	}

	if aiConfig.APIKey == "" {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "AI service not configured"})
		return
	}

	// Check cache for natural language query
	cacheKey := fmt.Sprintf("cache:natural:%s:%s", session.ID, hashString(request.Query))
	if cachedResult, err := rdb.Get(ctx, cacheKey).Result(); err == nil {
		var cachedResponse NaturalResponse
		if json.Unmarshal([]byte(cachedResult), &cachedResponse) == nil {
			cachedResponse.Cached = true
			c.JSON(http.StatusOK, cachedResponse)
			return
		}
	}

	// Get schema
	schema, err := getTableSchemaForSession(session.ID, session.TableName)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get table schema"})
		return
	}

	// Generate SQL using AI
	generatedSQL, err := generateSQL(request.Query, schema)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to generate SQL: %v", err)})
		return
	}

	// Execute the generated SQL
	start := time.Now()
	rows, err := db.Query(generatedSQL)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":         fmt.Sprintf("Failed to execute generated SQL: %v", err),
			"generated_sql": generatedSQL,
		})
		return
	}
	defer rows.Close()

	results, _ := processRows(rows)
	duration := time.Since(start)

	response := NaturalResponse{
		NaturalQuery: request.Query,
		GeneratedSQL: generatedSQL,
		Explanation:  fmt.Sprintf("Generated SQL query from natural language: '%s'", request.Query),
		Runtime:      duration.String(),
		Data:         results,
		RowCount:     len(results),
		SessionID:    session.ID,
		Cached:       false,
	}

	// Cache the result for 1 hour
	if responseData, err := json.Marshal(response); err == nil {
		rdb.Set(ctx, cacheKey, responseData, time.Hour)
	}

	c.JSON(http.StatusOK, response)
}

func handleSchema(c *gin.Context) {
	sessionID := c.Param("sessionId")

	session, err := getSession(sessionID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Invalid session: %v", err)})
		return
	}

	schema, err := getTableSchemaForSession(session.ID, session.TableName)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get table schema"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"schema":     schema,
		"session_id": session.ID,
	})
}

// Utility functions
func processRows(rows *sql.Rows) ([]map[string]interface{}, []string) {
	columns, _ := rows.Columns()
	var results []map[string]interface{}

	for rows.Next() {
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			continue
		}

		row := make(map[string]interface{})
		for i, col := range columns {
			val := values[i]
			if b, ok := val.([]byte); ok {
				row[col] = string(b)
			} else {
				row[col] = val
			}
		}
		results = append(results, row)
	}

	return results, columns
}

func getTableSchemaForSession(sessionID, tableName string) (string, error) {
	// Try cache first
	if cachedSchema, err := rdb.Get(ctx, fmt.Sprintf("schema:%s", sessionID)).Result(); err == nil {
		return cachedSchema, nil
	}

	// Check if table exists
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?", tableName).Scan(&count)
	if err != nil || count == 0 {
		return "No table found. Please upload a CSV file first.", nil
	}

	var schema strings.Builder
	schema.WriteString(fmt.Sprintf("Table: %s\n", tableName))

	columnRows, err := db.Query(fmt.Sprintf("PRAGMA table_info(%s)", tableName))
	if err != nil {
		return "", err
	}
	defer columnRows.Close()

	schema.WriteString("Columns:\n")
	for columnRows.Next() {
		var cid int
		var name, dataType string
		var notNull, pk int
		var defaultValue *string

		if err := columnRows.Scan(&cid, &name, &dataType, &notNull, &defaultValue, &pk); err != nil {
			continue
		}

		schema.WriteString(fmt.Sprintf("  - %s (%s)\n", name, dataType))
	}

	schemaStr := schema.String()
	// Cache the schema
	rdb.Set(ctx, fmt.Sprintf("schema:%s", sessionID), schemaStr, 24*time.Hour)

	return schemaStr, nil
}

func clearSessionCaches(sessionID string) {
	keys, _ := rdb.Keys(ctx, fmt.Sprintf("cache:*:%s:*", sessionID)).Result()
	if len(keys) > 0 {
		rdb.Del(ctx, keys...)
	}
	rdb.Del(ctx, fmt.Sprintf("schema:%s", sessionID))
}

func hashString(s string) string {
	// Simple hash function for cache keys
	hash := uint32(0)
	for _, c := range s {
		hash = hash*31 + uint32(c)
	}
	return fmt.Sprintf("%x", hash)
}

func checkRedisHealth() string {
	if _, err := rdb.Ping(ctx).Result(); err != nil {
		return "disconnected"
	}
	return "connected"
}

func cleanupExpiredSessions() {
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		// Get all session keys
		keys, err := rdb.Keys(ctx, "session:*").Result()
		if err != nil {
			continue
		}

		for _, key := range keys {
			sessionData, err := rdb.Get(ctx, key).Result()
			if err != nil {
				continue
			}

			var session Session
			if json.Unmarshal([]byte(sessionData), &session) != nil {
				continue
			}

			// Delete sessions not used for more than 24 hours
			if time.Since(session.LastUsed) > 24*time.Hour {
				sessionID := strings.TrimPrefix(key, "session:")

				// Clean up all related data
				pipe := rdb.Pipeline()
				pipe.Del(ctx, key)

				// Clean up cache keys
				cacheKeys, _ := rdb.Keys(ctx, fmt.Sprintf("cache:*:%s:*", sessionID)).Result()
				if len(cacheKeys) > 0 {
					pipe.Del(ctx, cacheKeys...)
				}

				pipe.Del(ctx, fmt.Sprintf("schema:%s", sessionID))
				pipe.Exec(ctx)

				// Drop the table
				tableName := fmt.Sprintf("data_%s", sessionID)
				db.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s", tableName))
			}
		}
	}
}

func generateSQL(query, schema string) (string, error) {
	prompt := fmt.Sprintf(`Convert this natural language query to SQL.

Database Schema:
%s

Natural Language Query: %s

You must respond with ONLY a JSON object in this exact format:
{"sql": "SELECT * FROM data WHERE..."}

Do not include any other text, explanations, or markdown formatting. Only return the JSON object.`, schema, query)

	aiReq := AIRequest{
		Model:     aiConfig.Model,
		MaxTokens: 1000,
		Messages: []Message{
			{
				Role:    "user",
				Content: prompt,
			},
		},
	}

	reqBody, err := json.Marshal(aiReq)
	if err != nil {
		return "", err
	}

	client := &http.Client{Timeout: 30 * time.Second}
	req, err := http.NewRequest("POST", aiConfig.BaseURL+"/v1/messages", bytes.NewBuffer(reqBody))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", aiConfig.APIKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var aiResp AIResponse
	if err := json.NewDecoder(resp.Body).Decode(&aiResp); err != nil {
		return "", err
	}

	if len(aiResp.Content) == 0 {
		return "", fmt.Errorf("empty response from AI")
	}

	var sqlResp SQLResponse
	if err := json.Unmarshal([]byte(aiResp.Content[0].Text), &sqlResp); err != nil {
		return "", fmt.Errorf("failed to parse AI response: %v", err)
	}

	return sqlResp.SQL, nil
}
