package model

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetCumulativeQuotaDataSumsHoursAndKeepsModelList(t *testing.T) {
	truncateTables(t)

	require.NoError(t, DB.Create(&QuotaData{
		UserID: 1, Username: "alice", ModelName: "gpt-a", ChannelID: 1, CreatedAt: 1500, Quota: 100, Count: 2, TokenUsed: 30, PromptTokens: 20, CompletionTokens: 10,
	}).Error)
	require.NoError(t, DB.Create(&QuotaData{
		UserID: 1, Username: "alice", ModelName: "gpt-a", ChannelID: 2, CreatedAt: 1500, Quota: 25, Count: 1, TokenUsed: 8, PromptTokens: 5, CompletionTokens: 3,
	}).Error)
	require.NoError(t, DB.Create(&QuotaData{
		UserID: 1, Username: "alice", ModelName: "gpt-b", CreatedAt: 1600, Quota: 40, Count: 4, TokenUsed: 50, PromptTokens: 40, CompletionTokens: 10,
	}).Error)
	require.NoError(t, DB.Create(&QuotaData{
		UserID: 1, Username: "alice", ModelName: "", CreatedAt: 1700, Quota: 9, Count: 1, TokenUsed: 7,
	}).Error)
	require.NoError(t, DB.Create(&QuotaData{
		UserID: 2, Username: "bob", ModelName: "gpt-a", CreatedAt: 1500, Quota: 70, Count: 3, TokenUsed: 12, PromptTokens: 9, CompletionTokens: 3,
	}).Error)
	require.NoError(t, DB.Create(&QuotaData{
		UserID: 1, Username: "alice", ModelName: "gpt-a", CreatedAt: 5000, Quota: 999,
	}).Error)

	all, err := GetCumulativeQuotaData(1000, 2000, 0, "", "")
	require.NoError(t, err)
	require.Equal(t, []string{"gpt-a", "gpt-b"}, all.Models)
	require.Equal(t, []HourlyQuotaPoint{
		{CreatedAt: 1500, Quota: 195, Count: 6, TokenUsed: 50, PromptTokens: 34, CompletionTokens: 16},
		{CreatedAt: 1600, Quota: 40, Count: 4, TokenUsed: 50, PromptTokens: 40, CompletionTokens: 10},
		{CreatedAt: 1700, Quota: 9, Count: 1, TokenUsed: 7},
	}, all.Points)

	filtered, err := GetCumulativeQuotaData(1000, 2000, 0, "", "gpt-a")
	require.NoError(t, err)
	require.Equal(t, []string{"gpt-a", "gpt-b"}, filtered.Models)
	require.Equal(t, []HourlyQuotaPoint{{CreatedAt: 1500, Quota: 195, Count: 6, TokenUsed: 50, PromptTokens: 34, CompletionTokens: 16}}, filtered.Points)

	alice, err := GetCumulativeQuotaData(1000, 2000, 1, "", "")
	require.NoError(t, err)
	require.Equal(t, []string{"gpt-a", "gpt-b"}, alice.Models)
	require.Equal(t, []HourlyQuotaPoint{
		{CreatedAt: 1500, Quota: 125, Count: 3, TokenUsed: 38, PromptTokens: 25, CompletionTokens: 13},
		{CreatedAt: 1600, Quota: 40, Count: 4, TokenUsed: 50, PromptTokens: 40, CompletionTokens: 10},
		{CreatedAt: 1700, Quota: 9, Count: 1, TokenUsed: 7},
	}, alice.Points)

	bob, err := GetCumulativeQuotaData(1000, 2000, 0, "bob", "")
	require.NoError(t, err)
	require.Equal(t, []string{"gpt-a"}, bob.Models)
	require.Equal(t, []HourlyQuotaPoint{{CreatedAt: 1500, Quota: 70, Count: 3, TokenUsed: 12, PromptTokens: 9, CompletionTokens: 3}}, bob.Points)
}

func TestLogQuotaDataAccumulatesPromptAndCompletionTokensAcrossFlushes(t *testing.T) {
	truncateTables(t)
	CacheQuotaDataLock.Lock()
	CacheQuotaData = make(map[string]*QuotaData)
	CacheQuotaDataLock.Unlock()

	params := QuotaDataLogParams{
		UserID: 1, Username: "alice", ModelName: "gpt-a", CreatedAt: 3700,
		Quota: 10, TokenUsed: 30, PromptTokens: 20, CompletionTokens: 10,
	}
	LogQuotaData(params)
	LogQuotaData(params)
	SaveQuotaDataCache()
	LogQuotaData(params)
	SaveQuotaDataCache()

	var rows []QuotaData
	require.NoError(t, DB.Find(&rows).Error)
	require.Len(t, rows, 1)
	require.Equal(t, 3, rows[0].Count)
	require.Equal(t, 90, rows[0].TokenUsed)
	require.Equal(t, 60, rows[0].PromptTokens)
	require.Equal(t, 30, rows[0].CompletionTokens)
}
