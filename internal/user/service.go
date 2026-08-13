package user

import (
	"fmt"

	"github.com/google/uuid"
)

// Service defines the business-logic contract for user operations.
type Service interface {
	// FindOrCreate fetches an existing user by phone or creates a new one.
	// This is called after OTP verification — the user is created on first login.
	FindOrCreate(phone, role, language string) (*User, bool, error)

	// GetByID fetches a user by their UUID, returns error if not found.
	GetByID(id uuid.UUID) (*User, error)

	// UpdateProfile updates the user's name and language preference.
	UpdateProfile(id uuid.UUID, name, language string) (*User, error)
}

type service struct {
	repo Repository
}

// NewService returns a Service backed by the given Repository.
func NewService(repo Repository) Service {
	return &service{repo: repo}
}

// FindOrCreate looks up a user by phone number.
// If no user exists, a new one is created with the provided role and language.
// Returns (user, isNew, error).
// isNew = true means this is the user's very first login.
func (s *service) FindOrCreate(phone, role, language string) (*User, bool, error) {
	// Validate role
	if !isValidRole(role) {
		return nil, false, fmt.Errorf("invalid role: %s", role)
	}

	// Validate language, default to English
	if language != LangFrench {
		language = LangEnglish
	}

	// Try to find existing user
	existing, err := s.repo.FindByPhone(phone)
	if err != nil {
		return nil, false, fmt.Errorf("FindOrCreate lookup: %w", err)
	}
	if existing != nil {
		return existing, false, nil
	}

	// First login — create new user
	newUser := &User{
		Phone:     phone,
		Role:      role,
		Language:  language,
		KYCStatus: KYCPending,
		IsActive:  true,
	}

	if err := s.repo.Create(newUser); err != nil {
		return nil, false, fmt.Errorf("FindOrCreate create: %w", err)
	}

	return newUser, true, nil
}

// GetByID fetches a user by UUID, returning an error if not found.
func (s *service) GetByID(id uuid.UUID) (*User, error) {
	u, err := s.repo.FindByID(id)
	if err != nil {
		return nil, err
	}
	if u == nil {
		return nil, fmt.Errorf("user not found")
	}
	return u, nil
}

// UpdateProfile changes the user's display name and language preference.
func (s *service) UpdateProfile(id uuid.UUID, name, language string) (*User, error) {
	u, err := s.GetByID(id)
	if err != nil {
		return nil, err
	}

	if name != "" {
		u.Name = name
	}
	if language == LangFrench || language == LangEnglish {
		u.Language = language
	}

	if err := s.repo.Update(u); err != nil {
		return nil, err
	}
	return u, nil
}

// isValidRole returns true if the given role string is one of the defined constants.
func isValidRole(role string) bool {
	switch role {
	case RoleShipper, RoleDriver, RoleFleetAdmin, RoleOwnerOperator, RoleAdmin:
		return true
	}
	return false
}
