package main

import (
	"bytes"
	"database/sql"
	"encoding/csv"
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
	"github.com/joho/godotenv"
)

var db *sql.DB

type AIConfig struct {
	APIKey  string
	BaseURL string
	Model   string
}

var aiConfig AIConfig

// request structs
type QueryRequest struct {
	SQL string `json:"sql"` //standard SQL
}

type NaturalRequest struct {
	Query string `json:"query"` // Natural language
}

// response structs
type QueryResponse struct {
	Data     []map[string]interface{} `json:"data"`
	Runtime  string                   `json:"runtime"`
	Query    string                   `json:"query"`
	RowCount int                      `json:"row_count"`
}

type NaturalResponse struct {
	NaturalQuery string                   `json:"natural_query"`
	GeneratedSQL string                   `json:"generated_sql"`
	Explanation  string                   `json:"explanation"`
	Runtime      string                   `json:"runtime"`
	Data         []map[string]interface{} `json:"data"`
	RowCount     int                      `json:"row_count"`
}

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

// Ai start
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

	// Verify API key is loaded
	if aiConfig.APIKey == "" {
		log.Println("Warning: AI_API_KEY not found in environment")
	} else {
		log.Println("AI configuration loaded successfully")
	}
}

func main() {
	//sql connection
	var err error
	db, err = sql.Open("sqlite", ":memory:")
	if err != nil {
		panic(err)
	}
	defer db.Close() //at end of program close db connection

	router := gin.Default()
	router.Use(cors.Default())

	//check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "OK",
		})
	})

	//upload
	router.POST("/api/upload", handleUpload)

	//quey
	router.POST("api/query", handleQuery)

	//natural language
	router.POST("/api/natural", handleNatural)

	//table schema
	router.GET("/api/schema", handleSchema)

	router.Run(":8080")
}

func handleUpload(c *gin.Context) {
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to get file"})
		return
	}
	defer file.Close()

	//read csv
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

	//create table
	headers := records[0]
	tableName := "data"

	db.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s", tableName))

	//Quote column names properly
	columns := make([]string, len(headers))
	for i, header := range headers {
		cleanHeader := strings.TrimSpace(header)
		columns[i] = fmt.Sprintf("`%s` TEXT", cleanHeader)
	}

	createSQL := fmt.Sprintf("CREATE TABLE %s (%s)", tableName, strings.Join(columns, ", "))

	fmt.Printf("Creating table with SQL: %s\n", createSQL)

	_, err = db.Exec(createSQL)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Failed to create table: %v", err),
			"sql":   createSQL,
		})
		return
	}

	//put in data
	for i, record := range records[1:] {
		if len(record) != len(headers) {
			fmt.Printf("Skipping row %d: column count mismatch\n", i+1)
			continue
		}

		placeholders := strings.Repeat("?,", len(record)-1) + "?"
		insertSQL := fmt.Sprintf("INSERT INTO %s VALUES (%s)", tableName, placeholders)

		values := make([]interface{}, len(record))
		for j, v := range record {
			values[j] = strings.TrimSpace(v)
		}

		_, err = db.Exec(insertSQL, values...)
		if err != nil {
			fmt.Printf("Error inserting row %d: %v\n", i+1, err)
			continue
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"message":  "File uploaded successfully",
		"filename": header.Filename,
		"rows":     len(records) - 1,
		"columns":  headers,
		"table":    tableName,
	})
}

func handleQuery(c *gin.Context) {
	start := time.Now()

	var request QueryRequest

	//try to read JSON of req and bind it to the request struct if error -> ...
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON"})
		return
	}

	//query to get row
	rows, err := db.Query(request.SQL)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to execute query: %v", err)})
		return
	}

	defer rows.Close() //close at end to STAY SAFE !!!!

	// Get column names
	columns, err := rows.Columns()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get columns"})
		return
	}

	//slice the row values
	var results []map[string]interface{} //interface to store any types :) //storing a list of rows
	for rows.Next() {
		//this part was super confusing to me so basically 1st values is to hold the values of the row and valuePtrs is to hold pointers to those values
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i] //HERE point each entry in valuePTRS to the corresponding entry in values
		}
		//"Please go fetch the values from the database, and store them in those locations."
		if err := rows.Scan(valuePtrs...); err != nil {
			continue
		}

		//mapping
		row := make(map[string]interface{})
		for i, col := range columns {
			val := values[i]
			if b, ok := val.([]byte); ok {
				row[col] = string(b) //convert byte slice to string
			} else {
				row[col] = val //store the value as is
			}
		}
		results = append(results, row)
	}
	duration := time.Since(start) //stop timer

	c.JSON(http.StatusOK, QueryResponse{
		Data:     results,
		Runtime:  fmt.Sprintf("%.3fms", float64(duration.Nanoseconds())/1e6),
		Query:    request.SQL,
		RowCount: len(results),
	})
}

func handleNatural(c *gin.Context) {
	overallStart := time.Now()
	var request NaturalRequest

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON"})
		return
	}

	// Check AI
	if aiConfig.APIKey == "" {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "AI service not configured"})
		return
	}

	// add schema context
	schema, err := getTableSchema()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get table schema"})
		return
	}
	// Generate SQL using AI
	aiStart := time.Now()
	generatedSQL, err := generateSQL(request.Query, schema)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to generate SQL: %v", err)})
		return
	}
	aiDuration := time.Since(aiStart)
	///run sql
	queryStart := time.Now()
	rows, err := db.Query(generatedSQL)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":         fmt.Sprintf("Failed to execute generated SQL: %v", err),
			"generated_sql": generatedSQL,
		})
		return
	}
	defer rows.Close()

	//similar running sql query similar to hte one above :)
	columns, err := rows.Columns()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get columns"})
		return
	}

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
				row[col] = string(b)			} else {
				row[col] = val
			}
		}
		results = append(results, row)
	}

	queryDuration := time.Since(queryStart)
	totalDuration := time.Since(overallStart)

	c.JSON(http.StatusOK, gin.H{
		"natural_query": request.Query,
		"generated_sql": generatedSQL,
		"explanation":   fmt.Sprintf("Generated SQL query from natural language: '%s'", request.Query),
		"data":          results,
		"row_count":     len(results),
		"timing": gin.H{
			"ai_generation_ms":   fmt.Sprintf("%.3f", float64(aiDuration.Nanoseconds())/1e6),
			"query_execution_ms": fmt.Sprintf("%.3f", float64(queryDuration.Nanoseconds())/1e6),
			"total_ms":          fmt.Sprintf("%.3f", float64(totalDuration.Nanoseconds())/1e6),
		},
	})
}

func handleSchema(c *gin.Context) {
	schema, err := getTableSchema()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get table schema"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"schema": schema,
	})
}

// get the table info for better ai contest
func getTableSchema() (string, error) {
	rows, err := db.Query("SELECT name FROM sqlite_master WHERE type='table' AND name != 'sqlite_sequence'")
	if err != nil {
		return "", err
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var tableName string
		if err := rows.Scan(&tableName); err != nil {
			continue
		}
		tables = append(tables, tableName)
	}

	if len(tables) == 0 {
		return "No tables found. Please upload a CSV file first.", nil
	}

	var schema strings.Builder
	for _, table := range tables {
		schema.WriteString(fmt.Sprintf("Table: %s\n", table))

		// Get column info
		columnRows, err := db.Query(fmt.Sprintf("PRAGMA table_info(%s)", table))
		if err != nil {
			continue
		}

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
		columnRows.Close()
		schema.WriteString("\n")
	}

	return schema.String(), nil
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

	// Parse the JSON response to extract SQL
	var sqlResp SQLResponse
	if err := json.Unmarshal([]byte(aiResp.Content[0].Text), &sqlResp); err != nil {
		return "", fmt.Errorf("failed to parse AI response: %v", err)
	}

	return sqlResp.SQL, nil
}
