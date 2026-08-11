package controllers

import (
    "net/http"
    "://github.com"
    "logistic-app-go/models"
)

// GetShipmentStatus handles fetching shipment info
func GetShipmentStatus(c *gin.Context) {
    id := c.Param("id")
    
    // Mock response data (Replace with database query later)
    shipment := models.Shipment{
        ID:          id,
        Origin:      "Accra",
        Destination: "Kumasi",
        Status:      "In Transit",
    }
    
    c.JSON(http.StatusOK, shipment)
}
