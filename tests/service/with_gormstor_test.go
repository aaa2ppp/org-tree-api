package service

import (
	"context"
	"fmt"
	"org-tree-api/internal/service"
	"org-tree-api/internal/storage/gormstor"
	"org-tree-api/internal/tests"
	"testing"

	"github.com/aaa2ppp/be"
	"github.com/pressly/goose/v3"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

const migrationsPath = "../../migrations"

func TestServiceWithGormStorage(t *testing.T) {
	ctx := context.Background()

	// 1. Поднимаем контейнер (один раз на все тесты)
	postgresC, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        "postgres:18.4-alpine",
			ExposedPorts: []string{"5432/tcp"},
			Env: map[string]string{
				"POSTGRES_USER":     "postgres",
				"POSTGRES_PASSWORD": "postgres",
				"POSTGRES_DB":       "company_tree_test",
			},
			WaitingFor: wait.ForLog("database system is ready to accept connections").WithOccurrence(2),
		},
		Started: true,
	})
	be.Err(t, err, nil)
	defer postgresC.Terminate(ctx)

	// 2. Подключаемся
	host, _ := postgresC.Host(ctx)
	port, _ := postgresC.MappedPort(ctx, "5432")
	dsn := fmt.Sprintf("host=%s port=%s user=postgres password=postgres dbname=company_tree_test sslmode=disable",
		host, port.Port())

	t.Logf("DSN: %q", dsn)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	be.Err(t, err, nil)
	sqlDB, err := db.DB()
	be.Err(t, err, nil)
	goose.SetDialect("postgres")

	newStorage := func(t *testing.T) service.Storage {
		err := goose.DownTo(sqlDB, migrationsPath, 0)
		be.Err(t, err, nil)
		err = goose.Up(sqlDB, migrationsPath)
		be.Err(t, err, nil)
		return gormstor.New(db)
	}

	t.Run("Create_Get_Departments", func(t *testing.T) {
		tests.Service_Create_Get_Departments(t, ctx, newStorage)
	})

	t.Run("Create_Get_Employees", func(t *testing.T) {
		tests.Service_Create_Get_Employees(t, ctx, newStorage)
	})

	t.Run("MoveDepartment", func(t *testing.T) {
		tests.Service_MoveDepartment(t, ctx, newStorage)
	})

	t.Run("DeleteDepartment", func(t *testing.T) {
		tests.Service_DeleteDepartment(t, ctx, newStorage)
	})
}
