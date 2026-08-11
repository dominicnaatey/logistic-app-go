package routes

import (
	"github.com/gin-gonic/gin"
	"logistic-app-go/controllers"
	"logistic-app-go/middleware"
	"logistic-app-go/repository"
	"logistic-app-go/services"
)

func SetupRouter() *gin.Engine {
	r := gin.New()

	// Global middleware
	r.Use(gin.Recovery())
	r.Use(middleware.Logger())
	r.Use(middleware.CORS())

	// Dependency injection
	shipmentRepo := repository.NewShipmentRepository()
	shipmentSvc := services.NewShipmentService(shipmentRepo)
	shipmentCtrl := controllers.NewShipmentController(shipmentSvc)

	// Versioned API grouping
	api := r.Group("/api/v1")
	{
		api.GET("/shipments/:id", shipmentCtrl.GetShipmentStatus)
		api.POST("/shipments", shipmentCtrl.CreateShipment)
		api.PUT("/shipments/:id", shipmentCtrl.UpdateShipment)
		api.DELETE("/shipments/:id", shipmentCtrl.DeleteShipment)
	}

	return r
}
