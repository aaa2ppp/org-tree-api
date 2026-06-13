package service_test

import (
	"org-tree-api/internal/service"
	"org-tree-api/internal/storage/memstor"
	"org-tree-api/internal/tests"

	"context"
	"testing"
)

func TestServiceWithMemStorsge(t *testing.T) {
	ctx := context.Background()
	newSrorage := func(_ *testing.T) service.Storage { return memstor.New() }

	t.Run("Create_Get_Departments", func(t *testing.T) {
		tests.Service_Create_Get_Departments(t, ctx, newSrorage)
	})

	t.Run("Create_Get_Employees", func(t *testing.T) {
		tests.Service_Create_Get_Employees(t, ctx, newSrorage)
	})

	t.Run("MoveDepartment", func(t *testing.T) {
		tests.Service_MoveDepartment(t, ctx, newSrorage)
	})

	t.Run("DeleteDepartment", func(t *testing.T) {
		tests.Service_DeleteDepartment(t, ctx, newSrorage)
	})
}
