package controller

import (
	"encoding/csv"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
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

// Admin export: all operational fields (matches admin usage-log table + diagnostics).
func writeAdminUsageLogsExportCSV(w *csv.Writer, logs []*model.Log, loc *time.Location) {
	header := []string{
		"日志ID", "时间", "日志类型", "用户ID", "用户名", "模型名称", "令牌名称",
		"输入Token数", "输出Token数", "cache_creation", "cache_read", "cache_write",
		"额度", "金额(USD)", "花费",
		"耗时(秒)", "是否流式", "渠道ID", "渠道名称", "分组", "IP", "请求ID", "日志内容", "其他信息",
	}
	_ = w.Write(header)
	for _, lg := range logs {
		ts := time.Unix(lg.CreatedAt, 0).In(loc).Format(time.RFC3339)
		quota := int64(lg.Quota)
		cache := parseUsageLogCacheCounts(lg.Other)
		_ = w.Write([]string{
			strconv.Itoa(lg.Id),
			ts,
			adminUserExportLogTypeName(lg.Type),
			strconv.Itoa(lg.UserId),
			lg.Username,
			lg.ModelName,
			lg.TokenName,
			strconv.Itoa(lg.PromptTokens),
			strconv.Itoa(lg.CompletionTokens),
			strconv.Itoa(cache.Creation),
			strconv.Itoa(cache.Read),
			strconv.Itoa(cache.Write),
			strconv.Itoa(lg.Quota),
			adminUserExportFormatAmount(adminUserExportAmountUSD(quota)),
			adminUserExportFormatAmount(adminUserExportDisplayAmount(quota)),
			strconv.Itoa(lg.UseTime),
			adminUserExportBoolText(lg.IsStream),
			strconv.Itoa(lg.ChannelId),
			lg.ChannelName,
			lg.Group,
			lg.Ip,
			lg.RequestId,
			lg.Content,
			lg.Other,
		})
	}
	w.Flush()
}

// User export: only fields visible on the non-admin usage-log page (both themes).
// Excludes channel, user identity, log id, raw other JSON, USD-only column, admin diagnostics.
func writeUserUsageLogsExportCSV(w *csv.Writer, logs []*model.Log, loc *time.Location) {
	header := []string{
		"时间", "日志类型", "令牌名称", "分组", "模型名称",
		"输入Token数", "输出Token数", "cache_creation", "cache_read", "cache_write",
		"额度", "花费",
		"耗时(秒)", "是否流式", "IP", "请求ID", "日志内容",
	}
	_ = w.Write(header)
	for _, lg := range logs {
		ts := time.Unix(lg.CreatedAt, 0).In(loc).Format(time.RFC3339)
		quota := int64(lg.Quota)
		cache := parseUsageLogCacheCounts(lg.Other)
		ip := ""
		if (lg.Type == model.LogTypeConsume || lg.Type == model.LogTypeError) && lg.Ip != "" {
			ip = lg.Ip
		}
		_ = w.Write([]string{
			ts,
			adminUserExportLogTypeName(lg.Type),
			lg.TokenName,
			lg.Group,
			lg.ModelName,
			strconv.Itoa(lg.PromptTokens),
			strconv.Itoa(lg.CompletionTokens),
			strconv.Itoa(cache.Creation),
			strconv.Itoa(cache.Read),
			strconv.Itoa(cache.Write),
			strconv.Itoa(lg.Quota),
			adminUserExportFormatAmount(adminUserExportDisplayAmount(quota)),
			strconv.Itoa(lg.UseTime),
			adminUserExportBoolText(lg.IsStream),
			ip,
			lg.RequestId,
			lg.Content,
		})
	}
	w.Flush()
}

func respondUsageLogsExport(c *gin.Context, filter model.LogListFilter) {
	logs, _, err := model.GetLogsForExport(filter, 0)
	if err != nil {
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
	filename := "usage-logs-" + time.Now().In(loc).Format("20060102-150405") + ".csv"
	adminUserExportSetCSVHeaders(c, filename)
	c.Status(http.StatusOK)

	if _, err = c.Writer.Write([]byte{0xEF, 0xBB, 0xBF}); err != nil {
		return
	}
	w := csv.NewWriter(c.Writer)
	if filter.ForAdmin {
		writeAdminUsageLogsExportCSV(w, logs, loc)
	} else {
		writeUserUsageLogsExportCSV(w, logs, loc)
	}
}

// ExportAllLogs exports filtered usage logs as CSV (admin).
func ExportAllLogs(c *gin.Context) {
	filter := parseLogExportFilter(c, 0, true)
	respondUsageLogsExport(c, filter)
}

// ExportUserLogs exports filtered usage logs as CSV for the current user.
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
