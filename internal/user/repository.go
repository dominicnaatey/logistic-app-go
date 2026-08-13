package user

import (
	"errors"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Repository defines the data-access contract for User records.
// Using an interface keeps the service layer decoupled from GORM,
// making it easy to swap implementations or mock in tests.
type Repository interface {
	FindByPhone(phone string) (*User, error)
	FindByID(id uuid.UUID) (*User, error)
	Create(u *User) error
	Update(u *User) error
}

// gormRepository is the production PostgreSQL-backed implementation.
type gormRepository struct {
	db *gorm.DB
}

// NewRepository returns a Repository backed by the given GORM connection.
func NewRepository(db *gorm.DB) Repository {
	return &gormRepository{db: db}
}

// FindByPhone looks up an active (non-deleted) user by their phone number.
// Returns (nil, nil) when the user does not exist so callers can distinguish
// "not found" from a real database error.
func (r *gormRepository) FindByPhone(phone string) (*User, error) {
	var u User
	err := r.db.Where("phone = ?", phone).First(&u).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil // not found is not an error at this layer
		}
		return nil, fmt.Errorf("FindByPhone: %w", err)
	}
	return &u, nil
}

// FindByID looks up an active user by their UUID primary key.
// Returns (nil, nil) when the user does not exist.
func (r *gormRepository) FindByID(id uuid.UUID) (*User, error) {
	var u User
	err := r.db.First(&u, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("FindByID: %w", err)
	}
	return &u, nil
}

// Create inserts a new user record.
// The User.BeforeCreate hook auto-assigns a UUID.
func (r *gormRepository) Create(u *User) error {
	if err := r.db.Create(u).Error; err != nil {
		return fmt.Errorf("Create user: %w", err)
	}
	return nil
}

// Update persists changes to an existing user record.
// Only non-zero fields are updated (GORM Save behaviour).
func (r *gormRepository) Update(u *User) error {
	if err := r.db.Save(u).Error; err != nil {
		return fmt.Errorf("Update user: %w", err)
	}
	return nil
}
