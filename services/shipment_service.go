package services

import (
	"logistic-app-go/models"
	"logistic-app-go/repository"
)

// ShipmentService defines the business logic interface for shipments.
type ShipmentService interface {
	GetShipmentByID(id string) (*models.Shipment, error)
	CreateShipment(origin, destination, status string) (*models.Shipment, error)
	UpdateShipment(id, origin, destination, status string) (*models.Shipment, error)
	DeleteShipment(id string) error
}

type shipmentService struct {
	repo repository.ShipmentRepository
}

// NewShipmentService creates a new ShipmentService instance.
func NewShipmentService(repo repository.ShipmentRepository) ShipmentService {
	return &shipmentService{repo: repo}
}

func (s *shipmentService) GetShipmentByID(id string) (*models.Shipment, error) {
	return s.repo.FindByID(id)
}

func (s *shipmentService) CreateShipment(origin, destination, status string) (*models.Shipment, error) {
	if status == "" {
		status = "Pending"
	}
	shipment := &models.Shipment{
		Origin:      origin,
		Destination: destination,
		Status:      status,
	}
	if err := s.repo.Create(shipment); err != nil {
		return nil, err
	}
	return shipment, nil
}

func (s *shipmentService) UpdateShipment(id, origin, destination, status string) (*models.Shipment, error) {
	shipment, err := s.repo.FindByID(id)
	if err != nil || shipment == nil {
		return nil, err
	}
	if origin != "" {
		shipment.Origin = origin
	}
	if destination != "" {
		shipment.Destination = destination
	}
	if status != "" {
		shipment.Status = status
	}
	if err := s.repo.Update(shipment); err != nil {
		return nil, err
	}
	return shipment, nil
}

func (s *shipmentService) DeleteShipment(id string) error {
	return s.repo.Delete(id)
}
