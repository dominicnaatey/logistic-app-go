package repository

import "logistic-app-go/models"

// ShipmentRepository defines the interface for shipment data access.
type ShipmentRepository interface {
	FindByID(id string) (*models.Shipment, error)
	Create(shipment *models.Shipment) error
	Update(shipment *models.Shipment) error
	Delete(id string) error
}

// shipmentRepo is the concrete implementation backed by a database.
type shipmentRepo struct {
	// TODO: add db client field, e.g. *mongo.Collection
}

// NewShipmentRepository creates a new ShipmentRepository instance.
func NewShipmentRepository() ShipmentRepository {
	return &shipmentRepo{}
}

func (r *shipmentRepo) FindByID(id string) (*models.Shipment, error) {
	// TODO: implement DB query
	return nil, nil
}

func (r *shipmentRepo) Create(shipment *models.Shipment) error {
	// TODO: implement DB insert
	return nil
}

func (r *shipmentRepo) Update(shipment *models.Shipment) error {
	// TODO: implement DB update
	return nil
}

func (r *shipmentRepo) Delete(id string) error {
	// TODO: implement DB delete
	return nil
}
