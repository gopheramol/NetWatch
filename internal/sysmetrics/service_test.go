package sysmetrics

import (
	"context"
	"testing"
	"time"

	"github.com/gopheramol/NetWatch/internal/database"
	"github.com/gopheramol/NetWatch/internal/repository"
	"github.com/gopheramol/NetWatch/internal/utils"
)

func TestSysMetricsCollectAndQuery(t *testing.T) {
	dbPath := t.TempDir() + "/test_sysmetrics.db"
	db, err := database.Open(dbPath)
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	defer db.Close()

	logger, _ := utils.NewLogger("debug", "")
	repo := repository.NewSystemMetricsRepository(db)
	svc := NewService(repo, nil, logger)

	ctx := context.Background()

	m, err := svc.Collect(ctx)
	if err != nil {
		t.Fatalf("failed to collect system metrics: %v", err)
	}
	if m == nil || m.ID == "" {
		t.Fatalf("expected collected metrics struct, got nil or empty ID")
	}

	latest, err := svc.Latest(ctx)
	if err != nil {
		t.Fatalf("failed to get latest metrics: %v", err)
	}
	if latest.ID != m.ID {
		t.Fatalf("expected latest ID %s, got %s", m.ID, latest.ID)
	}

	history, err := svc.History(ctx, time.Now().Add(-1*time.Hour), time.Now().Add(1*time.Hour), 10)
	if err != nil {
		t.Fatalf("failed to get history: %v", err)
	}
	if len(history) != 1 {
		t.Fatalf("expected 1 history record, got %d", len(history))
	}
}
