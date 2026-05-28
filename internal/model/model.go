package model

import "time"

type Department struct {
	ID        int
	Name      string
	ParentID  *int
	CreatedAt time.Time
	UpdatedAt time.Time
}

type Employee struct {
	ID           int
	DepartmentID int
	FullName     string
	Position     string
	HiredAt      *time.Time
	CreatedAt    time.Time
	UpdatedAt    time.Time
}
