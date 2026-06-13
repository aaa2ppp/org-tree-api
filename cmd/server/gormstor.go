//go:build !memstor

package main

import (
	"errors"
	"fmt"
	"log/slog"
	"net"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"org-tree-api/internal/config"
	"org-tree-api/internal/service"
	"org-tree-api/internal/storage/gormstor"
)

func openStorage(cfg config.Config) (service.Storage, error) {
	slog.Info("open database")
	db, err := openGORM(cfg.DB)
	if err != nil {
		slog.Error("open gorm", "error", err)
		return nil, errors.New("fatal")
	}
	return gormstor.New(db), nil
}

func openGORM(cfg config.DB) (*gorm.DB, error) {
	host, port, err := net.SplitHostPort(cfg.Addr)
	if err != nil {
		return nil, err
	}

	dsn := fmt.Sprintf("host=%s port=%s dbname=%s user=%s password=%s sslmode=%s",
		host, port, cfg.DBName, cfg.User, cfg.Password, cfg.SSLMode)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	return db, nil
}
