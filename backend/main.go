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

func main() {	//sql lite connection
	var err error
	db, err = sql.Open("sqlite", "./database.db")
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
	var request struct {
		SQL string `json:"sql"`
	}

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

	c.JSON(http.StatusOK, gin.H{
		"sql":       request.SQL,
		"columns":   columns,
		"results":   results,
		"row_count": len(results),
	})
}
