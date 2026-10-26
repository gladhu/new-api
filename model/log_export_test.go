package model

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func openLogExportTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	common.SetDatabaseTypes(common.DatabaseTypeSQLite, common.DatabaseTypeSQLite)
	InitColumnNames()
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

	logs, total, err := GetLogsForExport(context.Background(), LogListFilter{
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

	_, total, err = GetLogsForExport(context.Background(), LogListFilter{
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

func TestGetLogsForExportDetailedColumns(t *testing.T) {
	openLogExportTestDB(t)
	InitColumnNames()
	base := int64(1_700_000_000)
	seed := Log{
		UserId:            7,
		Username:          "alice",
		Type:              LogTypeConsume,
		ModelName:         "gpt-4",
		TokenName:         "tok",
		Content:           "prompt body",
		Other:             `{"cache_tokens":1}`,
		Quota:             10,
		CreatedAt:         base,
		Ip:                "10.0.0.8",
		RequestId:         "req-1",
		UpstreamRequestId: "up-1",
		Group:             "vip",
		UseTime:           3,
		IsStream:          true,
	}
	require.NoError(t, LOG_DB.Create(&seed).Error)

	filter := LogListFilter{
		UserId:         7,
		StartTimestamp: base,
		EndTimestamp:   base + 10,
		ForAdmin:       true,
	}
	slim, _, err := GetLogsForExport(context.Background(), filter, 0)
	require.NoError(t, err)
	require.Len(t, slim, 1)
	require.Empty(t, slim[0].Content)
	require.Empty(t, slim[0].Username)
	require.Empty(t, slim[0].Ip)
	require.Equal(t, "gpt-4", slim[0].ModelName)

	filter.Detailed = true
	detailed, _, err := GetLogsForExport(context.Background(), filter, 0)
	require.NoError(t, err)
	require.Len(t, detailed, 1)
	require.Equal(t, "prompt body", detailed[0].Content)
	require.Equal(t, "alice", detailed[0].Username)
	require.Equal(t, "10.0.0.8", detailed[0].Ip)
	require.Equal(t, "vip", detailed[0].Group)
	require.Equal(t, "req-1", detailed[0].RequestId)
	require.Equal(t, "up-1", detailed[0].UpstreamRequestId)
	require.True(t, detailed[0].IsStream)
	require.Equal(t, 3, detailed[0].UseTime)

	filter.ForAdmin = false
	hidden, _, err := GetLogsForExport(context.Background(), filter, 0)
	require.NoError(t, err)
	require.Len(t, hidden, 1)
	require.Empty(t, hidden[0].Content)
	require.Empty(t, hidden[0].Username)
}
