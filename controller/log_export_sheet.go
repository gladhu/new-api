package controller

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	"github.com/shopspring/decimal"
	"github.com/xuri/excelize/v2"
)

const usageLogXLSXContentType = "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"

// usageLogExportDownloadChunkBytes is the write/log step size during download.
// 1 MiB keeps progress logs useful without flooding the log file.
const usageLogExportDownloadChunkBytes = 1 << 20

var usageLogExportHeaders = []interface{}{
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
}

func writeUsageLogsXLSX(ctx context.Context, w io.Writer, logs []*model.Log, loc *time.Location) error {
	logger.LogInfo(ctx, fmt.Sprintf("usage log export stage=workbook begin rows=%d", len(logs)))
	buildStarted := time.Now()
	file, err := newUsageLogsWorkbook(logs, loc)
	logUsageExportStage(ctx, "workbook", len(logs), time.Since(buildStarted))
	if err != nil {
		return err
	}
	defer file.Close()
	buf, err := packUsageLogsXLSX(ctx, file, len(logs))
	if err != nil {
		return err
	}
	return sendPackedUsageLogsXLSX(ctx, w, buf, len(logs))
}

// packUsageLogsXLSX packs the workbook with excelize's default zip deflate.
// DefaultCompression makes large exports smaller than BestSpeed; the extra CPU
// is usually cheaper than the download time on slow client links.
func packUsageLogsXLSX(ctx context.Context, file *excelize.File, rows int) (*bytes.Buffer, error) {
	logger.LogInfo(ctx, fmt.Sprintf("usage log export stage=generate begin rows=%d", rows))
	generateStarted := time.Now()
	buf, err := file.WriteToBuffer()
	logUsageExportStage(ctx, "generate", rows, time.Since(generateStarted))
	return buf, err
}

func sendPackedUsageLogsXLSX(ctx context.Context, w io.Writer, buf *bytes.Buffer, rows int) error {
	total := int64(buf.Len())
	logger.LogInfo(ctx, fmt.Sprintf("usage log export stage=download begin rows=%d bytes=%d", rows, total))
	downloadStarted := time.Now()
	data := buf.Bytes()
	var written int64
	var lastLogged int64
	for offset := 0; offset < len(data); {
		end := offset + usageLogExportDownloadChunkBytes
		if end > len(data) {
			end = len(data)
		}
		n, err := w.Write(data[offset:end])
		written += int64(n)
		offset += n
		if flusher, ok := w.(http.Flusher); ok {
			flusher.Flush()
		}
		if written-lastLogged >= usageLogExportDownloadChunkBytes || offset >= len(data) || err != nil {
			elapsed := time.Since(downloadStarted)
			speed := int64(0)
			if elapsed > 0 {
				speed = written * int64(time.Second) / int64(elapsed)
			}
			logger.LogInfo(ctx, fmt.Sprintf(
				"usage log export stage=download_progress rows=%d written=%d total=%d elapsed=%s speed=%dB/s",
				rows, written, total, elapsed.Round(time.Millisecond), speed,
			))
			lastLogged = written
		}
		if err != nil {
			model.RecordUsageExportStage(ctx, "download", time.Since(downloadStarted))
			return err
		}
		if n == 0 {
			model.RecordUsageExportStage(ctx, "download", time.Since(downloadStarted))
			return io.ErrShortWrite
		}
	}
	elapsed := time.Since(downloadStarted)
	logger.LogInfo(ctx, fmt.Sprintf("usage log export stage=download rows=%d bytes=%d elapsed=%s", rows, written, elapsed.Round(time.Millisecond)))
	model.RecordUsageExportStage(ctx, "download", elapsed)
	return nil
}

func newUsageLogsWorkbook(logs []*model.Log, loc *time.Location) (*excelize.File, error) {
	if loc == nil {
		loc = time.Local
	}
	file := excelize.NewFile()
	const sheet = "usage"
	if err := file.SetSheetName("Sheet1", sheet); err != nil {
		file.Close()
		return nil, err
	}
	writer, err := file.NewStreamWriter(sheet)
	if err != nil {
		file.Close()
		return nil, err
	}
	if err = writer.SetRow("A1", usageLogExportHeaders); err != nil {
		file.Close()
		return nil, err
	}
	for i, lg := range logs {
		cell, cellErr := excelize.CoordinatesToCellName(1, i+2)
		if cellErr != nil {
			file.Close()
			return nil, cellErr
		}
		if err = writer.SetRow(cell, usageLogExportRow(lg, loc)); err != nil {
			file.Close()
			return nil, err
		}
	}
	if err = writer.Flush(); err != nil {
		file.Close()
		return nil, err
	}
	return file, nil
}

func usageLogExportRow(lg *model.Log, loc *time.Location) []interface{} {
	other, err := common.StrToMap(lg.Other)
	if err != nil || other == nil {
		other = map[string]interface{}{}
	}
	cache := usageLogCacheCountsFromMap(other)
	readTokens := cache.Read
	writeTokens := cache.Write
	imageTokens := usageLogOtherInt(other, "image_output")
	baseTokens := usageLogInfactPromptTokens(lg.PromptTokens, other, readTokens, writeTokens, imageTokens)

	modelRatio := usageLogOtherFloatOr(other, "model_ratio", 0)
	groupRatio := usageLogOtherFloatOr(other, "group_ratio", 1)
	completionRatio := usageLogOtherFloatOr(other, "completion_ratio", 1)
	cacheRatio := usageLogOtherFloatOr(other, "cache_ratio", 1)
	imageRatio := usageLogOtherFloatOr(other, "image_ratio", 1)
	modelPrice := usageLogOtherFloatOr(other, "model_price", 0)
	tiered := usageLogOtherString(other, "billing_mode") == "tiered_expr"

	var inputPrice, readPrice, writePrice, imagePrice, outputPrice, cachePrice float64
	if !tiered && modelPrice <= 0 {
		inputPrice = usageLogComponentUSD(baseTokens, 1, modelRatio, groupRatio)
		readPrice = usageLogComponentUSD(readTokens, cacheRatio, modelRatio, groupRatio)
		writePrice = usageLogCacheWriteUSD(other, writeTokens, modelRatio, groupRatio)
		imagePrice = usageLogComponentUSD(imageTokens, imageRatio, modelRatio, groupRatio)
		outputPrice = usageLogComponentUSD(lg.CompletionTokens, completionRatio, modelRatio, groupRatio)
		cachePrice = decimal.NewFromFloat(readPrice).Add(decimal.NewFromFloat(writePrice)).Round(6).InexactFloat64()
	}

	return []interface{}{
		time.Unix(lg.CreatedAt, 0).In(loc).Format("2006-01-02 15:04:05"),
		lg.ModelName,
		lg.TokenName,
		lg.PromptTokens,
		baseTokens,
		readTokens + writeTokens,
		writeTokens,
		readTokens,
		imageTokens,
		lg.CompletionTokens,
		lg.CompletionTokens,
		inputPrice,
		cachePrice,
		writePrice,
		readPrice,
		imagePrice,
		outputPrice,
		usageLogQuotaUSD(int64(lg.Quota)),
	}
}

func usageLogInfactPromptTokens(prompt int, other map[string]interface{}, readTokens, writeTokens, imageTokens int) int {
	base := prompt
	if !usageLogClaudeSemantic(other) && !usageLogLegacyClaudeDerived(other) {
		base -= readTokens
		creation := usageLogOtherInt(other, "cache_creation_tokens")
		if creation > 0 {
			base -= creation
		} else {
			base -= writeTokens
		}
	}
	base -= imageTokens
	if usageLogOtherBool(other, "audio_input_seperate_price") {
		base -= usageLogOtherInt(other, "audio_input_token_count")
	}
	if base < 0 {
		return 0
	}
	return base
}

func usageLogCacheWriteUSD(other map[string]interface{}, writeTokens int, modelRatio, groupRatio float64) float64 {
	creation := usageLogOtherInt(other, "cache_creation_tokens")
	write5m := usageLogOtherInt(other, "cache_creation_tokens_5m")
	write1h := usageLogOtherInt(other, "cache_creation_tokens_1h")
	creationRatio := usageLogOtherFloatOr(other, "cache_creation_ratio", 1)
	if usageLogClaudeSemantic(other) || usageLogLegacyClaudeDerived(other) {
		if creation == 0 && write5m == 0 && write1h == 0 {
			return usageLogComponentUSD(writeTokens, creationRatio, modelRatio, groupRatio)
		}
		remaining := creation - write5m - write1h
		if remaining < 0 {
			remaining = 0
		}
		ratio5m := usageLogOtherFloatOr(other, "cache_creation_ratio_5m", 0)
		ratio1h := usageLogOtherFloatOr(other, "cache_creation_ratio_1h", 0)
		weighted := decimal.NewFromInt(int64(remaining)).Mul(decimal.NewFromFloat(creationRatio))
		weighted = weighted.Add(decimal.NewFromInt(int64(write5m)).Mul(decimal.NewFromFloat(ratio5m)))
		weighted = weighted.Add(decimal.NewFromInt(int64(write1h)).Mul(decimal.NewFromFloat(ratio1h)))
		return usageLogDecimalUSD(weighted, modelRatio, groupRatio)
	}
	tokens := creation
	if tokens == 0 {
		tokens = writeTokens
	}
	return usageLogComponentUSD(tokens, creationRatio, modelRatio, groupRatio)
}

func usageLogComponentUSD(tokens int, typeRatio, modelRatio, groupRatio float64) float64 {
	if tokens == 0 || typeRatio == 0 || modelRatio == 0 || groupRatio == 0 {
		return 0
	}
	weighted := decimal.NewFromInt(int64(tokens)).Mul(decimal.NewFromFloat(typeRatio))
	return usageLogDecimalUSD(weighted, modelRatio, groupRatio)
}

func usageLogDecimalUSD(weightedTokens decimal.Decimal, modelRatio, groupRatio float64) float64 {
	if common.QuotaPerUnit <= 0 || weightedTokens.IsZero() || modelRatio == 0 || groupRatio == 0 {
		return 0
	}
	amount := weightedTokens.
		Mul(decimal.NewFromFloat(modelRatio)).
		Mul(decimal.NewFromFloat(groupRatio)).
		Div(decimal.NewFromFloat(common.QuotaPerUnit))
	return amount.Round(6).InexactFloat64()
}

func usageLogQuotaUSD(quota int64) float64 {
	if common.QuotaPerUnit <= 0 || quota == 0 {
		return 0
	}
	amount := decimal.NewFromInt(quota).Div(decimal.NewFromFloat(common.QuotaPerUnit))
	return amount.Round(6).InexactFloat64()
}

func usageLogClaudeSemantic(other map[string]interface{}) bool {
	if usageLogOtherBool(other, "claude") {
		return true
	}
	return usageLogOtherString(other, "usage_semantic") == "anthropic"
}

func usageLogLegacyClaudeDerived(other map[string]interface{}) bool {
	if usageLogClaudeSemantic(other) || usageLogOtherString(other, "usage_semantic") != "" {
		return false
	}
	return usageLogOtherInt(other, "cache_creation_tokens_5m") > 0 || usageLogOtherInt(other, "cache_creation_tokens_1h") > 0
}

func usageLogOtherBool(other map[string]interface{}, key string) bool {
	value, ok := other[key].(bool)
	return ok && value
}

func usageLogOtherString(other map[string]interface{}, key string) string {
	value, _ := other[key].(string)
	return value
}

func usageLogOtherFloatOr(other map[string]interface{}, key string, fallback float64) float64 {
	value, ok := other[key]
	if !ok || value == nil {
		return fallback
	}
	switch number := value.(type) {
	case float64:
		return number
	case float32:
		return float64(number)
	case int:
		return float64(number)
	case int32:
		return float64(number)
	case int64:
		return float64(number)
	case json.Number:
		parsed, err := number.Float64()
		if err != nil {
			return fallback
		}
		return parsed
	default:
		return fallback
	}
}
