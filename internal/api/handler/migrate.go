package handler

import (
	"context"
	"fmt"
	"log/slog"

	db "github.com/konfidence-project/konfidence/cmd/api/db"
	"github.com/konfidence-project/konfidence/internal/api/openapi"
)

type migrateHandler struct {
	dsn    string
	logger *slog.Logger
}

func newMigrateHandler(dsn string, logger *slog.Logger) *migrateHandler {
	return &migrateHandler{dsn: dsn, logger: logger}
}

func (h *migrateHandler) MigrateUpV1(
	ctx context.Context,
	_ openapi.MigrateUpV1RequestObject,
) (openapi.MigrateUpV1ResponseObject, error) {
	if h.dsn == "" {
		return nil, fmt.Errorf("migrations are not available: no database configured")
	}
	applied, err := db.MigrateUp(ctx, h.dsn, h.logger)
	if err != nil {
		return nil, fmt.Errorf("running migrations up: %w", err)
	}
	return openapi.MigrateUpV1200JSONResponse{Applied: applied}, nil
}

func (h *migrateHandler) MigrateDownV1(
	ctx context.Context,
	_ openapi.MigrateDownV1RequestObject,
) (openapi.MigrateDownV1ResponseObject, error) {
	if h.dsn == "" {
		return nil, fmt.Errorf("migrations are not available: no database configured")
	}
	applied, err := db.MigrateDown(ctx, h.dsn, h.logger)
	if err != nil {
		return nil, fmt.Errorf("running migration down: %w", err)
	}
	return openapi.MigrateDownV1200JSONResponse{Applied: applied}, nil
}
