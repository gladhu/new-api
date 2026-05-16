package controller

import (
	"encoding/csv"
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

func writeUsageLogsExportCSV(w *csv.Writer, logs []*model.Log, loc *time.Location) {
	header := []string{
		"日志ID", "时间", "日志类型", "用户ID", "用户名", "模型名称", "令牌名称",
		"输入Token数", "输出Token数", "额度", "金额(USD)", "当前展示金额",
		"耗时(秒)", "是否流式", "渠道ID", "渠道名称", "分组", "IP", "请求ID", "日志内容", "其他信息",
	}
	_ = w.Write(header)
	for _, lg := range logs {
		ts := time.Unix(lg.CreatedAt, 0).In(loc).Format(time.RFC3339)
		quota := int64(lg.Quota)
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
	writeUsageLogsExportCSV(w, logs, loc)
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
