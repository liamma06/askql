package main

import (
	"database/sql"
	"encoding/csv"
	"fmt"

	//"io"
	"net/http"
	"strings"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	_ "github.com/glebarez/go-sqlite"
)

var db *sql.DB

func main() {
	//sql lite connection
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
	//router.POST("api/query", handleQuery)

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

	// FIX: Quote column names properly
	columns := make([]string, len(headers))
	for i, header := range headers {
		cleanHeader := strings.TrimSpace(header)
		// Quote column names to handle special characters
		columns[i] = fmt.Sprintf("`%s` TEXT", cleanHeader)
	}

	createSQL := fmt.Sprintf("CREATE TABLE %s (%s)", tableName, strings.Join(columns, ", "))

	// DEBUG: Print the SQL to see what's being created
	fmt.Printf("Creating table with SQL: %s\n", createSQL)

	_, err = db.Exec(createSQL)
	if err != nil {
		// Better error message
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
			// Don't return error for individual row failures, just log
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
