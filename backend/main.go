package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
	//"github.com/gin-contrib/cors"
)

func main() {
	router := gin.Default()

	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "OK",
		})
	})

	router.POST("/api/upload", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "File uploaded successfully",
			"file":    "test.csv",
		})
	})

	router.Run(":8080")
}
