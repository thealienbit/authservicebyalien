package store

import (
	"time"

	"github.com/google/uuid"
)

const (
	AppStatusActive   = "ACTIVE"
	AppStatusDisabled = "DISABLED"
)

type App struct {
	ID        uuid.UUID
	Name      string
	BaseURL   string
	Status    string
	CreatedAt time.Time
	UpdatedAt time.Time
}

const (
	OrganizationStatusActive   = "ACTIVE"
	OrganizationStatusDisabled = "DISABLED"
)

type Organization struct {
	ID        uuid.UUID
	AppID     uuid.UUID
	Name      string
	Status    string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type Role string

const (
	RoleSuperAdmin Role = "SUPERADMIN"
	RoleAdmin      Role = "ADMIN"
	RoleClerk      Role = "CLERK"
)

func (r Role) Valid() bool {
	switch r {
	case RoleSuperAdmin, RoleAdmin, RoleClerk:
		return true
	default:
		return false
	}
}

type Membership struct {
	UserID         uuid.UUID
	OrganizationID uuid.UUID
	Role           Role
	CreatedAt      time.Time
	UpdatedAt      time.Time
}
