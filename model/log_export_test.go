package model

import (
	"fmt"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func openLogExportTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	common.UsingSQLite = true
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&Log{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	LOG_DB = db
	return db
}

func TestGetLogsForExport(t *testing.T) {
	openLogExportTestDB(t)
	base := int64(1_700_000_000)
	seed := []Log{
		{UserId: 1, Username: "alice", Type: LogTypeConsume, ModelName: "gpt-4", Quota: 10, CreatedAt: base},
		{UserId: 2, Username: "bob", Type: LogTypeConsume, ModelName: "gpt-4", Quota: 20, CreatedAt: base + 100},
		{UserId: 1, Username: "alice", Type: LogTypeTopup, Quota: 100, CreatedAt: base + 200},
	}
	if err := LOG_DB.Create(&seed).Error; err != nil {
		t.Fatal(err)
	}

	logs, total, err := GetLogsForExport(LogListFilter{
		LogType:        LogTypeConsume,
		StartTimestamp: base,
		EndTimestamp:   base + 1000,
		ForAdmin:       true,
	}, 0)
	if err != nil {
		t.Fatal(err)
	}
	if total != 2 || len(logs) != 2 {
		t.Fatalf("want 2 consume logs, got total=%d len=%d", total, len(logs))
	}

	_, total, err = GetLogsForExport(LogListFilter{
		UserId:         1,
		LogType:        LogTypeUnknown,
		StartTimestamp: base,
		EndTimestamp:   base + 1000,
	}, 0)
	if err != nil {
		t.Fatal(err)
	}
	if total != 2 {
		t.Fatalf("want 2 logs for user 1, got %d", total)
	}
}
