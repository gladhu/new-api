package model

import (
	"fmt"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func openLogConsumptionTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	common.UsingSQLite = true
	common.UsingMySQL = false
	common.UsingPostgreSQL = false

	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&Log{}); err != nil {
		t.Fatalf("migrate logs: %v", err)
	}
	LOG_DB = db
	t.Cleanup(func() {
		sqlDB, err := db.DB()
		if err == nil {
			_ = sqlDB.Close()
		}
	})
	return db
}

func TestSumChannelConsumption(t *testing.T) {
	openLogConsumptionTestDB(t)

	base := int64(1_700_000_000)
	logs := []Log{
		{UserId: 1, Username: "alice", Type: LogTypeConsume, ChannelId: 10, Quota: 100, PromptTokens: 10, CompletionTokens: 20, CreatedAt: base},
		{UserId: 1, Username: "alice", Type: LogTypeConsume, ChannelId: 10, Quota: 50, PromptTokens: 5, CompletionTokens: 5, CreatedAt: base + 100},
		{UserId: 2, Username: "bob", Type: LogTypeConsume, ChannelId: 10, Quota: 200, PromptTokens: 30, CompletionTokens: 40, CreatedAt: base + 200},
		{UserId: 2, Username: "bob", Type: LogTypeConsume, ChannelId: 11, Quota: 999, PromptTokens: 1, CompletionTokens: 1, CreatedAt: base + 200},
		{UserId: 1, Username: "alice", Type: LogTypeTopup, ChannelId: 10, Quota: 500, CreatedAt: base + 300},
		{UserId: 1, Username: "alice", Type: LogTypeConsume, ChannelId: 10, Quota: 300, PromptTokens: 1, CompletionTokens: 1, CreatedAt: base + 400*24*3600},
	}
	if err := LOG_DB.Create(&logs).Error; err != nil {
		t.Fatalf("seed logs: %v", err)
	}

	start := base
	end := base + 24*3600

	all, err := SumChannelConsumption(10, 0, "", start, end)
	if err != nil {
		t.Fatalf("sum channel: %v", err)
	}
	if all.Quota != 350 || all.RequestCount != 3 || all.PromptTokens != 45 || all.CompletionTokens != 65 {
		t.Fatalf("unexpected channel aggregate: %+v", all)
	}

	alice, err := SumChannelConsumption(10, 1, "", start, end)
	if err != nil {
		t.Fatalf("sum user channel: %v", err)
	}
	if alice.Quota != 150 || alice.RequestCount != 2 {
		t.Fatalf("unexpected user aggregate: %+v", alice)
	}

	byName, err := SumChannelConsumption(10, 0, "bob", start, end)
	if err != nil {
		t.Fatalf("sum by username: %v", err)
	}
	if byName.Quota != 200 {
		t.Fatalf("unexpected username aggregate: %+v", byName)
	}

	if _, err := SumChannelConsumption(10, 0, "", end+1, start); err == nil {
		t.Fatal("expected invalid range error")
	}
}
