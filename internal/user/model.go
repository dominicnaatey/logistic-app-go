package user

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Role constants define all valid user roles in the system.
// These map to the different actors in the logistics platform.
const (
	RoleShipper       = "shipper"        // Posts loads for transport
	RoleDriver        = "driver"         // Drives trucks, works for fleet or independent
	RoleFleetAdmin    = "fleet_admin"    // Manages a fleet company (owner or manager)
	RoleOwnerOperator = "owner_operator" // Independent truck owner-driver
	RoleAdmin         = "admin"          // Platform administrator
)

// KYCStatus constants define the possible states of identity verification.
const (
	KYCPending  = "pending"
	KYCApproved = "approved"
	KYCRejected = "rejected"
)

// Language constants define supported UI languages.
// Mali uses French (fr), Ghana uses English (en).
const (
	LangEnglish = "en"
	LangFrench  = "fr"
)

// User represents a registered user of the logistics platform.
// A user can have one of several roles that determine their
// capabilities and what screens they see in the app.
type User struct {
	// ID is a UUID primary key — avoids sequential ID enumeration attacks
	ID uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`

	// Phone is the primary identifier for login (E.164 format, e.g. +233501234567)
	Phone string `gorm:"type:varchar(20);uniqueIndex;not null" json:"phone"`

	// Role determines what the user can do in the system
	Role string `gorm:"type:varchar(50);not null" json:"role"`

	// Name is the user's full name (optional at registration, required before first transaction)
	Name string `gorm:"type:varchar(255)" json:"name"`

	// Language preference for SMS and push notifications (en | fr)
	Language string `gorm:"type:varchar(10);not null;default:'en'" json:"language"`

	// KYCStatus tracks identity verification state
	// Drivers and fleet admins must be approved before accepting loads
	KYCStatus string `gorm:"type:varchar(50);not null;default:'pending'" json:"kyc_status"`

	// IsActive allows soft-disabling accounts without deletion
	IsActive bool `gorm:"not null;default:true" json:"is_active"`

	// CreatedAt / UpdatedAt / DeletedAt managed by GORM automatically
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"` // soft delete
}

// BeforeCreate is a GORM hook that runs before inserting a new User.
// It auto-generates a UUID if one hasn't been set.
func (u *User) BeforeCreate(tx *gorm.DB) error {
	if u.ID == uuid.Nil {
		u.ID = uuid.New()
	}
	return nil
}

// IsDriver returns true if the user is a driver or owner-operator.
// Both roles can accept and execute trips.
func (u *User) IsDriver() bool {
	return u.Role == RoleDriver || u.Role == RoleOwnerOperator
}

// IsFleetManager returns true if the user manages a fleet company.
func (u *User) IsFleetManager() bool {
	return u.Role == RoleFleetAdmin
}

// CanAcceptLoads returns true if the user's role and KYC status
// allow them to accept trip assignments.
func (u *User) CanAcceptLoads() bool {
	return u.IsDriver() && u.KYCStatus == KYCApproved && u.IsActive
}

// IsKYCApproved returns true if the user has passed identity verification.
func (u *User) IsKYCApproved() bool {
	return u.KYCStatus == KYCApproved
}

// TableName explicitly sets the table name to avoid any GORM pluralisation
// ambiguity.
func (u *User) TableName() string {
	return "users"
}
