package main

import (
    "github.com/gin-gonic/gin"
)

func main() {
    // Create a default Gin router
    r := gin.Default()

    // Define a test route
    r.GET("/ping", func(c *gin.Context) {
        c.JSON(200, gin.H{
            "message": "pong",
        })
    })

    // Run the server (listens on port 8080 by default)
    r.Run() 
}
