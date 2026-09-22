package controller

import (
	"bytes"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/stretchr/testify/require"
	"github.com/xuri/excelize/v2"
)

func TestWriteUsageLogsXLSX_slimColumns(t *testing.T) {
	oldQuota := common.QuotaPerUnit
	common.QuotaPerUnit = 500000
	t.Cleanup(func() { common.QuotaPerUnit = oldQuota })

	var buf bytes.Buffer
	logs := []*model.Log{
		{
			Id:               99,
			UserId:           1,
			Username:         "alice",
			Type:             model.LogTypeConsume,
			CreatedAt:        1700000000,
			TokenName:        "tok",
			Group:            "default",
			ModelName:        "gpt-4",
			PromptTokens:     1000,
			CompletionTokens: 200,
			Quota:            12345,
			ChannelName:      "secret-channel",
			Ip:               "1.2.3.4",
			Content:          "should-not-appear",
			Other: `{
				"model_ratio": 1,
				"group_ratio": 1,
				"completion_ratio": 2,
				"cache_ratio": 0.5,
				"cache_creation_ratio": 1.25,
				"cache_tokens": 100,
				"cache_creation_tokens": 200,
				"cache_write_tokens": 200,
				"image_output": 50,
				"image_ratio": 2,
				"model_price": 0
			}`,
		},
	}
	require.NoError(t, writeUsageLogsXLSX(&buf, logs, time.UTC))

	file, err := excelize.OpenReader(bytes.NewReader(buf.Bytes()))
	require.NoError(t, err)
	t.Cleanup(func() { _ = file.Close() })

	rows, err := file.GetRows("usage")
	require.NoError(t, err)
	require.Len(t, rows, 2)
	require.Equal(t, []string{
		"created_time",
		"model_name",
		"token_name",
		"prompt_tokens",
		"infact_prompt_tokens",
		"cached_tokens",
		"cached_write_tokens",
		"cached_read_tokens",
		"output_image_tokens",
		"completion_tokens",
		"infact_completion_tokens",
		"input_price",
		"cached_tokens_price",
		"cached_write_tokens_price",
		"cached_read_tokens_price",
		"output_image_price",
		"output_price",
		"cost",
	}, rows[0])

	joined := rows[1]
	require.NotContains(t, joined, "secret-channel")
	require.NotContains(t, joined, "alice")
	require.NotContains(t, joined, "should-not-appear")
	require.Equal(t, "2023-11-14 22:13:20", joined[0])
	require.Equal(t, "gpt-4", joined[1])
	require.Equal(t, "tok", joined[2])
	require.Equal(t, "1000", joined[3])
	require.Equal(t, "650", joined[4])
	require.Equal(t, "300", joined[5])
	require.Equal(t, "200", joined[6])
	require.Equal(t, "100", joined[7])
	require.Equal(t, "50", joined[8])
	require.Equal(t, "200", joined[9])
	require.Equal(t, "200", joined[10])
	require.Equal(t, "0.0013", joined[11])
	require.Equal(t, "0.0006", joined[12])
	require.Equal(t, "0.0005", joined[13])
	require.Equal(t, "0.0001", joined[14])
	require.Equal(t, "0.0002", joined[15])
	require.Equal(t, "0.0008", joined[16])
	require.Equal(t, "0.02469", joined[17])
}

func TestWriteUsageLogsXLSX_claudeKeepsPromptAndPricesCacheWriteSplit(t *testing.T) {
	oldQuota := common.QuotaPerUnit
	common.QuotaPerUnit = 500000
	t.Cleanup(func() { common.QuotaPerUnit = oldQuota })

	row := usageLogExportRow(&model.Log{
		CreatedAt:        1700000000,
		ModelName:        "claude-sonnet",
		TokenName:        "tok",
		PromptTokens:     500,
		CompletionTokens: 20,
		Quota:            500000,
		Other: `{
			"claude": true,
			"model_ratio": 1,
			"group_ratio": 2,
			"completion_ratio": 5,
			"cache_ratio": 0.1,
			"cache_creation_ratio": 1.25,
			"cache_creation_ratio_5m": 1.25,
			"cache_creation_ratio_1h": 2,
			"cache_tokens": 80,
			"cache_creation_tokens": 100,
			"cache_creation_tokens_5m": 40,
			"cache_creation_tokens_1h": 60
		}`,
	}, time.UTC)

	require.Equal(t, 500, row[4])
	require.Equal(t, 180, row[5])
	require.Equal(t, 100, row[6])
	require.Equal(t, 80, row[7])
	// write weighted = 40*1.25 + 60*2 = 170; USD = 170 * 1 * 2 / 500000
	require.Equal(t, 0.00068, row[13])
	// read = 80 * 0.1 * 1 * 2 / 500000
	require.Equal(t, 0.000032, row[14])
	require.Equal(t, 0.000712, row[12])
	require.Equal(t, 1.0, row[17])
}

func TestWriteUsageLogsXLSX_perCallAndTieredOmitComponentPrices(t *testing.T) {
	oldQuota := common.QuotaPerUnit
	common.QuotaPerUnit = 500000
	t.Cleanup(func() { common.QuotaPerUnit = oldQuota })

	perCall := usageLogExportRow(&model.Log{
		PromptTokens:     10,
		CompletionTokens: 5,
		Quota:            250000,
		Other:            `{"model_price": 0.5, "model_ratio": 1, "group_ratio": 1}`,
	}, time.UTC)
	require.Equal(t, 0.0, perCall[11])
	require.Equal(t, 0.0, perCall[16])
	require.Equal(t, 0.5, perCall[17])

	tiered := usageLogExportRow(&model.Log{
		PromptTokens: 100,
		Quota:        1000,
		Other:        `{"billing_mode":"tiered_expr","model_ratio":1,"group_ratio":1,"completion_ratio":1}`,
	}, time.UTC)
	require.Equal(t, 0.0, tiered[11])
	require.Equal(t, 0.002, tiered[17])
}

func TestParseUsageLogCacheCounts(t *testing.T) {
	t.Run("explicit write tokens", func(t *testing.T) {
		got := parseUsageLogCacheCounts(`{"cache_tokens":80,"cache_creation_tokens":120,"cache_write_tokens":150}`)
		require.Equal(t, usageLogCacheCounts{Creation: 120, Read: 80, Write: 150}, got)
	})
	t.Run("falls back to creation when write missing", func(t *testing.T) {
		got := parseUsageLogCacheCounts(`{"cache_tokens":10,"cache_creation_tokens":40}`)
		require.Equal(t, usageLogCacheCounts{Creation: 40, Read: 10, Write: 40}, got)
	})
	t.Run("uses 5m and 1h split when write missing", func(t *testing.T) {
		got := parseUsageLogCacheCounts(`{"cache_creation_tokens_5m":20,"cache_creation_tokens_1h":30}`)
		require.Equal(t, usageLogCacheCounts{Creation: 0, Read: 0, Write: 50}, got)
	})
	t.Run("prefers larger creation over split sum", func(t *testing.T) {
		got := parseUsageLogCacheCounts(`{"cache_creation_tokens":80,"cache_creation_tokens_5m":20,"cache_creation_tokens_1h":30}`)
		require.Equal(t, usageLogCacheCounts{Creation: 80, Read: 0, Write: 80}, got)
	})
	t.Run("empty other", func(t *testing.T) {
		got := parseUsageLogCacheCounts("{}")
		require.Equal(t, usageLogCacheCounts{}, got)
	})
}
