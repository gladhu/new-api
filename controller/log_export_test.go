package controller

import (
	"bytes"
	"encoding/csv"
	"strings"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/model"
	"github.com/stretchr/testify/require"
)

func TestWriteUserUsageLogsExportCSV_omitsAdminFields(t *testing.T) {
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
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
			PromptTokens:     10,
			CompletionTokens: 20,
			Quota:            100,
			UseTime:          3,
			IsStream:         true,
			ChannelId:        5,
			ChannelName:      "secret-channel",
			Ip:               "1.2.3.4",
			RequestId:        "req-1",
			Content:          "ok",
			Other:            "{}",
		},
	}
	writeUserUsageLogsExportCSV(w, logs, time.UTC)
	w.Flush()

	rows, err := csv.NewReader(strings.NewReader(buf.String())).ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 {
		t.Fatalf("want header+1 row, got %d", len(rows))
	}
	header := strings.Join(rows[0], ",")
	if strings.Contains(header, "渠道") || strings.Contains(header, "用户") || strings.Contains(header, "其他信息") {
		t.Fatalf("user header must not contain admin columns: %v", rows[0])
	}
	row := strings.Join(rows[1], ",")
	if strings.Contains(row, "secret-channel") || strings.Contains(row, "alice") {
		t.Fatalf("user row leaked admin-only data: %s", row)
	}
	if !strings.Contains(row, "tok") || !strings.Contains(row, "gpt-4") {
		t.Fatalf("user row missing visible fields: %s", row)
	}
}

func TestWriteUserUsageLogsExportCSV_includesCacheColumns(t *testing.T) {
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	logs := []*model.Log{
		{
			Type:             model.LogTypeConsume,
			CreatedAt:        1700000000,
			TokenName:        "tok",
			Group:            "default",
			ModelName:        "claude-sonnet",
			PromptTokens:     1000,
			CompletionTokens: 50,
			Quota:            10,
			Other:            `{"cache_tokens":80,"cache_creation_tokens":120,"cache_write_tokens":120}`,
		},
	}
	writeUserUsageLogsExportCSV(w, logs, time.UTC)
	w.Flush()

	rows, err := csv.NewReader(strings.NewReader(buf.String())).ReadAll()
	require.NoError(t, err)
	require.Len(t, rows, 2)

	creationIdx := csvHeaderIndex(t, rows[0], "cache_creation")
	readIdx := csvHeaderIndex(t, rows[0], "cache_read")
	writeIdx := csvHeaderIndex(t, rows[0], "cache_write")
	require.Equal(t, "120", rows[1][creationIdx])
	require.Equal(t, "80", rows[1][readIdx])
	require.Equal(t, "120", rows[1][writeIdx])
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

func TestWriteAdminUsageLogsExportCSV_includesCacheColumns(t *testing.T) {
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	logs := []*model.Log{
		{
			Id:               1,
			UserId:           2,
			Username:         "bob",
			Type:             model.LogTypeConsume,
			CreatedAt:        1700000000,
			ModelName:        "gpt-4",
			PromptTokens:     10,
			CompletionTokens: 5,
			Quota:            1,
			Other:            `{"cache_tokens":3,"cache_creation_tokens":4,"cache_write_tokens":4}`,
		},
	}
	writeAdminUsageLogsExportCSV(w, logs, time.UTC)
	w.Flush()

	rows, err := csv.NewReader(strings.NewReader(buf.String())).ReadAll()
	require.NoError(t, err)
	require.Len(t, rows, 2)
	require.Equal(t, "4", rows[1][csvHeaderIndex(t, rows[0], "cache_creation")])
	require.Equal(t, "3", rows[1][csvHeaderIndex(t, rows[0], "cache_read")])
	require.Equal(t, "4", rows[1][csvHeaderIndex(t, rows[0], "cache_write")])
}

func csvHeaderIndex(t *testing.T, header []string, name string) int {
	t.Helper()
	for i, col := range header {
		if col == name {
			return i
		}
	}
	t.Fatalf("header missing %q: %v", name, header)
	return -1
}
