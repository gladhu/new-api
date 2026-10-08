package model

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func resetQuotaDataCache(t *testing.T) {
	t.Helper()
	CacheQuotaDataLock.Lock()
	CacheQuotaData = make(map[string]*QuotaData)
	CacheQuotaDataLock.Unlock()
	t.Cleanup(func() {
		CacheQuotaDataLock.Lock()
		CacheQuotaData = make(map[string]*QuotaData)
		CacheQuotaDataLock.Unlock()
	})
}

func closedQuotaHour(t *testing.T) int64 {
	t.Helper()
	now := time.Now().Unix()
	return now - now%3600 - 2*3600
}

func insertConsumeLog(t *testing.T, createdAt int64, quota int, prompt int, completion int) {
	t.Helper()
	require.NoError(t, LOG_DB.Create(&Log{
		UserId:           1,
		Username:         "alice",
		CreatedAt:        createdAt,
		Type:             LogTypeConsume,
		ModelName:        "gpt-5.6-sol",
		Quota:            quota,
		PromptTokens:     prompt,
		CompletionTokens: completion,
		TokenId:          11,
		ChannelId:        1,
		Group:            "vip",
	}).Error)
}

func sumQuotaData(t *testing.T) (count int, quota int, tokenUsed int) {
	t.Helper()
	var rows []QuotaData
	require.NoError(t, DB.Find(&rows).Error)
	for _, row := range rows {
		count += row.Count
		quota += row.Quota
		tokenUsed += row.TokenUsed
	}
	return count, quota, tokenUsed
}

func TestSaveQuotaDataCacheKeepsQuotaWhenWriteFails(t *testing.T) {
	truncateTables(t)
	resetQuotaDataCache(t)

	LogQuotaData(QuotaDataLogParams{
		UserID:    1,
		Username:  "alice",
		ModelName: "gpt-5.6-sol",
		CreatedAt: closedQuotaHour(t) + 10,
		UseGroup:  "vip",
		TokenID:   11,
		ChannelID: 1,
		NodeName:  "node-a",
		Quota:     40,
		TokenUsed: 8,
	})

	original := writeQuotaDataRow
	writeQuotaDataRow = func(*QuotaData) error { return errors.New("db down") }
	t.Cleanup(func() { writeQuotaDataRow = original })

	SaveQuotaDataCache()

	CacheQuotaDataLock.Lock()
	require.Len(t, CacheQuotaData, 1)
	for _, row := range CacheQuotaData {
		require.Equal(t, 40, row.Quota)
		require.Equal(t, 1, row.Count)
	}
	CacheQuotaDataLock.Unlock()

	var stored int64
	require.NoError(t, DB.Model(&QuotaData{}).Count(&stored).Error)
	require.Zero(t, stored)

	writeQuotaDataRow = original
	SaveQuotaDataCache()
	count, quota, tokenUsed := sumQuotaData(t)
	require.Equal(t, 1, count)
	require.Equal(t, 40, quota)
	require.Equal(t, 8, tokenUsed)
}

func TestSaveQuotaDataCacheAddsToExistingHour(t *testing.T) {
	truncateTables(t)
	resetQuotaDataCache(t)
	hour := closedQuotaHour(t)

	LogQuotaData(QuotaDataLogParams{
		UserID: 1, Username: "alice", ModelName: "gpt-5.6-sol", CreatedAt: hour + 10,
		UseGroup: "vip", TokenID: 11, ChannelID: 1, NodeName: "node-a", Quota: 100, TokenUsed: 20,
	})
	SaveQuotaDataCache()
	LogQuotaData(QuotaDataLogParams{
		UserID: 1, Username: "alice", ModelName: "gpt-5.6-sol", CreatedAt: hour + 20,
		UseGroup: "vip", TokenID: 11, ChannelID: 1, NodeName: "node-a", Quota: 50, TokenUsed: 5,
	})
	SaveQuotaDataCache()

	count, quota, tokenUsed := sumQuotaData(t)
	require.Equal(t, 2, count)
	require.Equal(t, 150, quota)
	require.Equal(t, 25, tokenUsed)
	CacheQuotaDataLock.Lock()
	require.Empty(t, CacheQuotaData)
	CacheQuotaDataLock.Unlock()
}

func TestReconcileQuotaDataFromLogsFillsMissingQuotaAndIsIdempotent(t *testing.T) {
	truncateTables(t)
	resetQuotaDataCache(t)
	hour := closedQuotaHour(t)
	insertConsumeLog(t, hour+10, 100, 30, 10)
	insertConsumeLog(t, hour+20, 50, 10, 5)
	require.NoError(t, DB.Create(&QuotaData{
		UserID: 1, Username: "alice", ModelName: "gpt-5.6-sol", CreatedAt: hour,
		UseGroup: "vip", TokenID: 11, ChannelID: 1, NodeName: "node-a",
		Count: 1, Quota: 100, TokenUsed: 20,
	}).Error)

	require.NoError(t, ReconcileQuotaDataFromLogs(hour, hour+3600))
	count, quota, tokenUsed := sumQuotaData(t)
	require.Equal(t, 2, count)
	require.Equal(t, 150, quota)
	require.Equal(t, 55, tokenUsed)

	require.NoError(t, ReconcileQuotaDataFromLogs(hour, hour+3600))
	count, quota, tokenUsed = sumQuotaData(t)
	require.Equal(t, 2, count)
	require.Equal(t, 150, quota)
	require.Equal(t, 55, tokenUsed)
}

func TestReconcileQuotaDataFromLogsRemovesExtraQuota(t *testing.T) {
	truncateTables(t)
	resetQuotaDataCache(t)
	hour := closedQuotaHour(t)
	insertConsumeLog(t, hour+10, 150, 20, 5)
	require.NoError(t, DB.Create(&QuotaData{
		UserID: 1, Username: "alice", ModelName: "gpt-5.6-sol", CreatedAt: hour,
		UseGroup: "vip", TokenID: 11, ChannelID: 1, NodeName: "node-a",
		Count: 2, Quota: 200, TokenUsed: 40,
	}).Error)

	require.NoError(t, ReconcileQuotaDataFromLogs(hour, hour+3600))
	count, quota, tokenUsed := sumQuotaData(t)
	require.Equal(t, 1, count)
	require.Equal(t, 150, quota)
	require.Equal(t, 25, tokenUsed)
}

func TestReconcileQuotaDataFromLogsSkipsCachedAndOpenHour(t *testing.T) {
	truncateTables(t)
	resetQuotaDataCache(t)
	hour := closedQuotaHour(t)
	insertConsumeLog(t, hour+10, 100, 10, 2)
	LogQuotaData(QuotaDataLogParams{
		UserID: 1, Username: "alice", ModelName: "gpt-5.6-sol", CreatedAt: hour + 10,
		UseGroup: "vip", TokenID: 11, ChannelID: 1, NodeName: "node-a", Quota: 100, TokenUsed: 12,
	})

	require.NoError(t, ReconcileQuotaDataFromLogs(hour, hour+7200))
	_, quota, _ := sumQuotaData(t)
	require.Zero(t, quota)

	now := time.Now().Unix()
	openHour := now - now%3600
	insertConsumeLog(t, openHour+5, 80, 4, 1)
	require.NoError(t, ReconcileQuotaDataFromLogs(openHour, now+60))
	_, quota, _ = sumQuotaData(t)
	require.Zero(t, quota)

	SaveQuotaDataCache()
	require.NoError(t, ReconcileQuotaDataFromLogs(hour, hour+7200))
	count, quota, tokenUsed := sumQuotaData(t)
	require.Equal(t, 1, count)
	require.Equal(t, 100, quota)
	require.Equal(t, 12, tokenUsed)
}
