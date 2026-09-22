package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"

	"github.com/gin-gonic/gin"
)

func parseLogExportFilter(c *gin.Context, userId int, forAdmin bool) model.LogListFilter {
	logType, _ := strconv.Atoi(c.Query("type"))
	startTimestamp, _ := strconv.ParseInt(c.Query("start_timestamp"), 10, 64)
	endTimestamp, _ := strconv.ParseInt(c.Query("end_timestamp"), 10, 64)
	channel, _ := strconv.Atoi(c.Query("channel"))
	return model.LogListFilter{
		UserId:         userId,
		LogType:        logType,
		StartTimestamp: startTimestamp,
		EndTimestamp:   endTimestamp,
		ModelName:      c.Query("model_name"),
		Username:       c.Query("username"),
		TokenName:      c.Query("token_name"),
		ChannelId:      channel,
		Group:          c.Query("group"),
		RequestId:      c.Query("request_id"),
		ForAdmin:       forAdmin,
	}
}

func parseLogExportLocation(c *gin.Context) *time.Location {
	loc := time.Local
	if tz := strings.TrimSpace(c.Query("timezone")); tz != "" {
		if parsed, err := time.LoadLocation(tz); err == nil {
			loc = parsed
		}
	}
	return loc
}

func respondUsageLogsExport(c *gin.Context, filter model.LogListFilter) {
	started := time.Now()
	logger.LogInfo(c, fmt.Sprintf("usage log export stage=start user_id=%d type=%d", filter.UserId, filter.LogType))
	logs, _, err := model.GetLogsForExport(c, filter, 0)
	if err != nil {
		logUsageExportStage(c, "done", 0, time.Since(started))
		if strings.Contains(err.Error(), "导出上限") {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": err.Error(),
			})
			return
		}
		common.ApiError(c, err)
		return
	}

	loc := parseLogExportLocation(c)
	buildStarted := time.Now()
	file, err := newUsageLogsWorkbook(logs, loc)
	logUsageExportStage(c, "workbook", len(logs), time.Since(buildStarted))
	if err != nil {
		logUsageExportStage(c, "done", len(logs), time.Since(started))
		common.ApiError(c, err)
		return
	}
	defer file.Close()

	buf, err := packUsageLogsXLSX(c, file, len(logs))
	if err != nil {
		logUsageExportStage(c, "done", len(logs), time.Since(started))
		common.ApiError(c, err)
		return
	}
	filename := "usage-logs-" + time.Now().In(loc).Format("20060102-150405") + ".xlsx"
	adminUserExportSetDownloadHeaders(c, usageLogXLSXContentType, filename)
	c.Status(http.StatusOK)
	err = sendPackedUsageLogsXLSX(c, c.Writer, buf, len(logs))
	logUsageExportStage(c, "done", len(logs), time.Since(started))
	if err != nil {
		logger.LogInfo(c, "usage log export stage=download error="+err.Error())
	}
}

func logUsageExportStage(ctx context.Context, stage string, rows int, elapsed time.Duration) {
	logger.LogInfo(ctx, fmt.Sprintf("usage log export stage=%s rows=%d elapsed=%s", stage, rows, elapsed.Round(time.Millisecond)))
}

// ExportAllLogs exports filtered usage logs as Excel (admin).
func ExportAllLogs(c *gin.Context) {
	filter := parseLogExportFilter(c, 0, true)
	respondUsageLogsExport(c, filter)
}

// ExportUserLogs exports filtered usage logs as Excel for the current user.
func ExportUserLogs(c *gin.Context) {
	userId := c.GetInt("id")
	filter := parseLogExportFilter(c, userId, false)
	respondUsageLogsExport(c, filter)
}

type usageLogCacheCounts struct {
	Creation int
	Read     int
	Write    int
}

// parseUsageLogCacheCounts reads cache token fields from a usage-log Other JSON.
// cache_read comes from cache_tokens; cache_creation from cache_creation_tokens;
// cache_write prefers cache_write_tokens, then the 5m/1h split, then cache_creation.
func parseUsageLogCacheCounts(otherJSON string) usageLogCacheCounts {
	other, err := common.StrToMap(otherJSON)
	if err != nil || other == nil {
		return usageLogCacheCounts{}
	}
	return usageLogCacheCountsFromMap(other)
}

func usageLogCacheCountsFromMap(other map[string]interface{}) usageLogCacheCounts {
	creation := usageLogOtherInt(other, "cache_creation_tokens")
	read := usageLogOtherInt(other, "cache_tokens")
	write := usageLogOtherInt(other, "cache_write_tokens")
	if write == 0 {
		write5m := usageLogOtherInt(other, "cache_creation_tokens_5m")
		write1h := usageLogOtherInt(other, "cache_creation_tokens_1h")
		if write5m > 0 || write1h > 0 {
			write = write5m + write1h
			if creation > write {
				write = creation
			}
		} else {
			write = creation
		}
	}
	return usageLogCacheCounts{Creation: creation, Read: read, Write: write}
}

func usageLogOtherInt(other map[string]interface{}, key string) int {
	v, ok := other[key]
	if !ok || v == nil {
		return 0
	}
	switch n := v.(type) {
	case int:
		return n
	case int32:
		return int(n)
	case int64:
		return int(n)
	case float64:
		return int(n)
	case json.Number:
		i, err := n.Int64()
		if err != nil {
			return 0
		}
		return int(i)
	default:
		return 0
	}
}
