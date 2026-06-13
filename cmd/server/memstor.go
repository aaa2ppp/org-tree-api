//go:build memstor

package main

import (
	"log/slog"
	"org-tree-api/internal/config"
	"org-tree-api/internal/service"
	"org-tree-api/internal/storage/memstor"
)

func newStorage(cfg config.Config) (service.Storage, error) {
	slog.Info("create storage in memory")
	return memstor.New(), nil
}
