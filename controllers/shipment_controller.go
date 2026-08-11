package controllers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"logistic-app-go/services"
)

// ShipmentController handles HTTP requests for shipments.
type ShipmentController struct {
	service services.ShipmentService
}

// NewShipmentController creates a new ShipmentController instance.
func NewShipmentController(s services.ShipmentService) *ShipmentController {
	return &ShipmentController{service: s}
}

// GetShipmentStatus godoc
// GET /api/v1/shipments/:id
func (ctrl *ShipmentController) GetShipmentStatus(c *gin.Context) {
	id := c.Param("id")

	shipment, err := ctrl.service.GetShipmentByID(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch shipment"})
		return
	}
	if shipment == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "shipment not found"})
		return
	}

	c.JSON(http.StatusOK, shipment)
}

// CreateShipment godoc
// POST /api/v1/shipments
func (ctrl *ShipmentController) CreateShipment(c *gin.Context) {
	var input struct {
		Origin      string `json:"origin" binding:"required"`
		Destination string `json:"destination" binding:"required"`
		Status      string `json:"status"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	shipment, err := ctrl.service.CreateShipment(input.Origin, input.Destination, input.Status)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create shipment"})
		return
	}

	c.JSON(http.StatusCreated, shipment)
}

// UpdateShipment godoc
// PUT /api/v1/shipments/:id
func (ctrl *ShipmentController) UpdateShipment(c *gin.Context) {
	id := c.Param("id")

	var input struct {
		Origin      string `json:"origin"`
		Destination string `json:"destination"`
		Status      string `json:"status"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	shipment, err := ctrl.service.UpdateShipment(id, input.Origin, input.Destination, input.Status)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update shipment"})
		return
	}
	if shipment == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "shipment not found"})
		return
	}

	c.JSON(http.StatusOK, shipment)
}

// DeleteShipment godoc
// DELETE /api/v1/shipments/:id
func (ctrl *ShipmentController) DeleteShipment(c *gin.Context) {
	id := c.Param("id")

	if err := ctrl.service.DeleteShipment(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete shipment"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "shipment deleted"})
}
