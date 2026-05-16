package controller

import (
	"bytes"
	"encoding/csv"
	"strings"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/model"
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
