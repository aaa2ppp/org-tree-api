package model

import (
	"math"
	"time"
)

const (
	MaxID             = math.MaxInt32
	VirtualRoot       = -1
	MaxNameLength     = 200
	MaxFullNameLength = 200
	MaxPositionLength = 200
	MaxDepth          = 5
)

func ValidID[T int | int32 | int64](id T) bool {
	return 0 < id && id <= MaxID
}

type Department struct {
	ID        int       `json:"id,omitempty"`
	Name      string    `json:"name,omitempty"`
	ParentID  int       `json:"parent_id,omitempty"`
	CreatedAt time.Time `json:"created_at,omitzero" format:"date-time" example:"2026-05-31T14:17:10.0+03:00"`
	UpdatedAt time.Time `json:"updated_at,omitzero" format:"date-time" example:"2026-05-31T14:17:10.0+03:00"`
}

type Employee struct {
	ID           int       `json:"id,omitempty"`
	DepartmentID int       `json:"department_id,omitempty"`
	FullName     string    `json:"full_name,omitempty"`
	Position     string    `json:"position,omitempty"`
	HiredAt      Date      `json:"hired_at,omitzero" swaggertype:"string" format:"date" example:"2026-05-31"`
	CreatedAt    time.Time `json:"created_at,omitzero" format:"date-time" example:"2026-05-31T14:17:10.0+03:00"`
	UpdatedAt    time.Time `json:"updated_at,omitzero" format:"date-time" example:"2026-05-31T14:17:10.0+03:00"`
}

//go:generate enumer -type=SortBy -trimprefix=SortBy -transform=snake
type SortBy uint8

const (
	SortByID SortBy = iota
	SortByName
	SortByCreatedAt
)

type GetDepartmentsTreeRequest struct {
	ID               int
	Depth            int
	IncludeEmployees bool
	SortBy           SortBy
}

type DepartmentNode struct {
	Department *Department       `json:"department,omitempty"`
	Employees  []*Employee       `json:"employees,omitempty"`
	Children   []*DepartmentNode `json:"children,omitempty"`
}

type MoveDepartmentRequest struct {
	ID       int
	Name     string
	ParentID int
}

//go:generate enumer -type=DeleteMode -trimprefix=DeleteMode -transform=snake
type DeleteMode uint8

const (
	_ DeleteMode = iota
	DeleteModeCascade
	DeleteModeReassign
)

type DeleteDepartmentRequest struct {
	ID                     int
	Mode                   DeleteMode
	ReassignToDepartmentID int
}
