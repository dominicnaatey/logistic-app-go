package routes

import (
    "github.com/gin-gonic/gin"
    "logistic-app-go/controllers"
)

func SetupRouter() *gin.Engine {
    r := gin.Default()

    // Versioned API grouping
    api := r.Group("/api/v1")
    {
        api.GET("/shipments/:id", controllers.GetShipmentStatus)
    }

    return r
}
