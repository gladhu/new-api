package controller

import (
	"archive/zip"
	"encoding/csv"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/operation_setting"

	"github.com/gin-gonic/gin"
)

func GetAllLogs(c *gin.Context) {
	pageInfo := common.GetPageQuery(c)
	logType, _ := strconv.Atoi(c.Query("type"))
	startTimestamp, _ := strconv.ParseInt(c.Query("start_timestamp"), 10, 64)
	endTimestamp, _ := strconv.ParseInt(c.Query("end_timestamp"), 10, 64)
	username := c.Query("username")
	tokenName := c.Query("token_name")
	modelName := c.Query("model_name")
	channel, _ := strconv.Atoi(c.Query("channel"))
	group := c.Query("group")
	requestId := c.Query("request_id")
	logs, total, err := model.GetAllLogs(logType, startTimestamp, endTimestamp, modelName, username, tokenName, pageInfo.GetStartIdx(), pageInfo.GetPageSize(), channel, group, requestId)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(logs)
	common.ApiSuccess(c, pageInfo)
	return
}

func GetUserLogs(c *gin.Context) {
	pageInfo := common.GetPageQuery(c)
	userId := c.GetInt("id")
	logType, _ := strconv.Atoi(c.Query("type"))
	startTimestamp, _ := strconv.ParseInt(c.Query("start_timestamp"), 10, 64)
	endTimestamp, _ := strconv.ParseInt(c.Query("end_timestamp"), 10, 64)
	tokenName := c.Query("token_name")
	modelName := c.Query("model_name")
	group := c.Query("group")
	requestId := c.Query("request_id")
	logs, total, err := model.GetUserLogs(userId, logType, startTimestamp, endTimestamp, modelName, tokenName, pageInfo.GetStartIdx(), pageInfo.GetPageSize(), group, requestId)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(logs)
	common.ApiSuccess(c, pageInfo)
	return
}

// Deprecated: SearchAllLogs 已废弃，前端未使用该接口。
func SearchAllLogs(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"success": false,
		"message": "该接口已废弃",
	})
}

// Deprecated: SearchUserLogs 已废弃，前端未使用该接口。
func SearchUserLogs(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"success": false,
		"message": "该接口已废弃",
	})
}

func GetLogByKey(c *gin.Context) {
	tokenId := c.GetInt("token_id")
	if tokenId == 0 {
		c.JSON(200, gin.H{
			"success": false,
			"message": "无效的令牌",
		})
		return
	}
	logs, err := model.GetLogByTokenId(tokenId)
	if err != nil {
		c.JSON(200, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	c.JSON(200, gin.H{
		"success": true,
		"message": "",
		"data":    logs,
	})
}

func GetLogsStat(c *gin.Context) {
	logType, _ := strconv.Atoi(c.Query("type"))
	startTimestamp, _ := strconv.ParseInt(c.Query("start_timestamp"), 10, 64)
	endTimestamp, _ := strconv.ParseInt(c.Query("end_timestamp"), 10, 64)
	tokenName := c.Query("token_name")
	username := c.Query("username")
	modelName := c.Query("model_name")
	channel, _ := strconv.Atoi(c.Query("channel"))
	group := c.Query("group")
	stat, err := model.SumUsedQuota(logType, startTimestamp, endTimestamp, modelName, username, tokenName, channel, group)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	//tokenNum := model.SumUsedToken(logType, startTimestamp, endTimestamp, modelName, username, "")
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data": gin.H{
			"quota": stat.Quota,
			"rpm":   stat.Rpm,
			"tpm":   stat.Tpm,
		},
	})
	return
}

func GetLogsSelfStat(c *gin.Context) {
	username := c.GetString("username")
	logType, _ := strconv.Atoi(c.Query("type"))
	startTimestamp, _ := strconv.ParseInt(c.Query("start_timestamp"), 10, 64)
	endTimestamp, _ := strconv.ParseInt(c.Query("end_timestamp"), 10, 64)
	tokenName := c.Query("token_name")
	modelName := c.Query("model_name")
	channel, _ := strconv.Atoi(c.Query("channel"))
	group := c.Query("group")
	quotaNum, err := model.SumUsedQuota(logType, startTimestamp, endTimestamp, modelName, username, tokenName, channel, group)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	//tokenNum := model.SumUsedToken(logType, startTimestamp, endTimestamp, modelName, username, tokenName)
	c.JSON(200, gin.H{
		"success": true,
		"message": "",
		"data": gin.H{
			"quota": quotaNum.Quota,
			"rpm":   quotaNum.Rpm,
			"tpm":   quotaNum.Tpm,
			//"token": tokenNum,
		},
	})
	return
}

func DeleteHistoryLogs(c *gin.Context) {
	targetTimestamp, _ := strconv.ParseInt(c.Query("target_timestamp"), 10, 64)
	if targetTimestamp == 0 {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "target timestamp is required",
		})
		return
	}
	count, err := model.DeleteOldLog(c.Request.Context(), targetTimestamp, 100)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    count,
	})
	return
}

func parseAdminUserExportMonthQuery(c *gin.Context) (userId int, year int, month int, loc *time.Location, err error) {
	userId, err = strconv.Atoi(c.Query("user_id"))
	if err != nil || userId <= 0 {
		return 0, 0, 0, nil, errors.New("用户ID无效")
	}
	year, err = strconv.Atoi(c.Query("year"))
	if err != nil {
		return 0, 0, 0, nil, errors.New("年份无效")
	}
	month, err = strconv.Atoi(c.Query("month"))
	if err != nil {
		return 0, 0, 0, nil, errors.New("月份无效")
	}
	loc = time.Local
	if tz := strings.TrimSpace(c.Query("timezone")); tz != "" {
		loc, err = time.LoadLocation(tz)
		if err != nil {
			return 0, 0, 0, nil, errors.New("时区无效")
		}
	}
	return userId, year, month, loc, nil
}

func adminUserExportUsername(userId int) string {
	user, userErr := model.GetUserById(userId, false)
	if userErr == nil && user != nil {
		return user.Username
	}
	username, _ := model.GetUsernameById(userId, true)
	return username
}

func adminUserExportSafeFilenamePart(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "unknown"
	}
	value = strings.Map(func(r rune) rune {
		if r < 32 || r == 127 || strings.ContainsRune(`/\:*?"<>|`, r) {
			return '_'
		}
		return r
	}, value)
	if value == "" {
		return "unknown"
	}
	return value
}

func adminUserExportASCIIHeaderFilename(value string) string {
	value = strings.Map(func(r rune) rune {
		if r < 32 || r == 127 || r > 126 || strings.ContainsRune(`/\:*?"<>|`, r) {
			return '_'
		}
		return r
	}, value)
	if strings.Trim(value, "_") == "" {
		return "export.csv"
	}
	return value
}

func adminUserExportFilename(prefix string, userId int, username string, year, month int) string {
	return fmt.Sprintf("%s-user-%d-%s-%04d-%02d.csv", prefix, userId, adminUserExportSafeFilenamePart(username), year, month)
}

func adminUserExportSetCSVHeaders(c *gin.Context, filename string) {
	adminUserExportSetDownloadHeaders(c, "text/csv; charset=utf-8", filename)
}

func adminUserExportSetZipHeaders(c *gin.Context, filename string) {
	adminUserExportSetDownloadHeaders(c, "application/zip", filename)
}

func adminUserExportSetDownloadHeaders(c *gin.Context, contentType string, filename string) {
	asciiFilename := adminUserExportASCIIHeaderFilename(filename)
	c.Header("Content-Type", contentType)
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"; filename*=UTF-8''%s`, asciiFilename, url.PathEscape(filename)))
}

func adminUserExportAmountUSD(quota int64) float64 {
	if common.QuotaPerUnit <= 0 {
		return 0
	}
	return float64(quota) / common.QuotaPerUnit
}

func adminUserExportDisplayAmount(quota int64) float64 {
	amount := adminUserExportAmountUSD(quota)
	switch operation_setting.GetQuotaDisplayType() {
	case operation_setting.QuotaDisplayTypeCNY:
		return amount * operation_setting.USDExchangeRate
	case operation_setting.QuotaDisplayTypeCustom:
		return amount * operation_setting.GetUsdToCurrencyRate(operation_setting.USDExchangeRate)
	case operation_setting.QuotaDisplayTypeTokens:
		return float64(quota)
	default:
		return amount
	}
}

func adminUserExportFormatAmount(amount float64) string {
	return strconv.FormatFloat(amount, 'f', 6, 64)
}

func adminUserExportBoolText(v bool) string {
	if v {
		return "是"
	}
	return "否"
}

func adminUserExportDisplayCurrencyName() string {
	switch operation_setting.GetQuotaDisplayType() {
	case operation_setting.QuotaDisplayTypeCNY:
		return "人民币(CNY)"
	case operation_setting.QuotaDisplayTypeCustom:
		symbol := operation_setting.GetCurrencySymbol()
		if symbol == "" {
			symbol = "自定义货币"
		}
		return "自定义货币(" + symbol + ")"
	case operation_setting.QuotaDisplayTypeTokens:
		return "额度单位/Token"
	default:
		return "美元(USD)"
	}
}

func adminUserExportLogTypeName(logType int) string {
	switch logType {
	case model.LogTypeTopup:
		return "充值"
	case model.LogTypeConsume:
		return "消费"
	case model.LogTypeManage:
		return "管理"
	case model.LogTypeSystem:
		return "系统"
	case model.LogTypeError:
		return "错误"
	case model.LogTypeRefund:
		return "退款"
	default:
		return "未知"
	}
}

func writeAdminUserMonthlyBillCSV(w *csv.Writer, userId int, username string, year int, month int, loc *time.Location, startSec int64, endSec int64, typeRows []model.AdminUserMonthLogTypeAgg, modelRows []model.AdminUserMonthModelAgg) {
	displayCurrencyName := adminUserExportDisplayCurrencyName()
	_ = w.Write([]string{"分区", "用户ID", "用户名", "年份", "月份", "时区", "开始时间(Unix秒)", "结束时间(Unix秒)", "当前展示金额类型"})
	_ = w.Write([]string{"导出信息", strconv.Itoa(userId), username, strconv.Itoa(year), strconv.Itoa(month), loc.String(), strconv.FormatInt(startSec, 10), strconv.FormatInt(endSec, 10), displayCurrencyName})
	_ = w.Write([]string{})
	_ = w.Write([]string{"按日志类型汇总", "日志类型编码", "日志类型", "记录数", "额度合计", "折算金额(USD)", "当前展示金额"})
	for _, row := range typeRows {
		_ = w.Write([]string{
			"按日志类型汇总",
			strconv.Itoa(row.Type),
			adminUserExportLogTypeName(row.Type),
			strconv.FormatInt(row.Cnt, 10),
			strconv.FormatInt(row.QuotaSum, 10),
			adminUserExportFormatAmount(adminUserExportAmountUSD(row.QuotaSum)),
			adminUserExportFormatAmount(adminUserExportDisplayAmount(row.QuotaSum)),
		})
	}
	_ = w.Write([]string{})
	_ = w.Write([]string{"按模型消费汇总", "模型名称", "请求数", "消耗额度合计", "消耗金额(USD)", "当前展示消耗金额", "输入Token合计", "输出Token合计"})
	for _, row := range modelRows {
		_ = w.Write([]string{
			"按模型消费汇总",
			row.ModelName,
			strconv.FormatInt(row.Cnt, 10),
			strconv.FormatInt(row.QuotaSum, 10),
			adminUserExportFormatAmount(adminUserExportAmountUSD(row.QuotaSum)),
			adminUserExportFormatAmount(adminUserExportDisplayAmount(row.QuotaSum)),
			strconv.FormatInt(row.PromptSum, 10),
			strconv.FormatInt(row.CompletionSum, 10),
		})
	}
	w.Flush()
}

func writeAdminUserConsumptionDetailsCSV(w *csv.Writer, logs []*model.Log, loc *time.Location) {
	header := []string{"日志ID", "消费时间", "用户ID", "用户名", "模型名称", "令牌名称", "输入Token数", "输出Token数", "消耗额度", "消耗金额(USD)", "当前展示消耗金额", "耗时(秒)", "是否流式", "渠道ID", "渠道名称", "分组", "IP", "请求ID", "日志内容", "其他信息"}
	_ = w.Write(header)
	for _, lg := range logs {
		ts := time.Unix(lg.CreatedAt, 0).In(loc).Format(time.RFC3339)
		quota := int64(lg.Quota)
		_ = w.Write([]string{
			strconv.Itoa(lg.Id),
			ts,
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

// ExportAdminUserMonthlyBill CSV: 月账单摘要（按日志类型汇总 + 按模型消费汇总）。
func ExportAdminUserMonthlyBill(c *gin.Context) {
	userId, year, month, loc, err := parseAdminUserExportMonthQuery(c)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	startSec, endSec, err := model.AdminUserMonthRangeSeconds(year, month, loc)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}

	username := adminUserExportUsername(userId)

	typeRows, err := model.GetUserLogTypeAggregatesForRange(userId, startSec, endSec)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	modelRows, err := model.GetUserConsumeSummaryByModelForRange(userId, startSec, endSec)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	filename := adminUserExportFilename("monthly-bill", userId, username, year, month)
	adminUserExportSetCSVHeaders(c, filename)
	c.Status(http.StatusOK)

	if _, err = c.Writer.Write([]byte{0xEF, 0xBB, 0xBF}); err != nil {
		return
	}
	w := csv.NewWriter(c.Writer)
	writeAdminUserMonthlyBillCSV(w, userId, username, year, month, loc, startSec, endSec, typeRows, modelRows)
}

// ExportAdminUserConsumptionDetails CSV: 指定自然月内该用户的消费（调用）明细。
func ExportAdminUserConsumptionDetails(c *gin.Context) {
	userId, year, month, loc, err := parseAdminUserExportMonthQuery(c)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	startSec, endSec, err := model.AdminUserMonthRangeSeconds(year, month, loc)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}

	logs, err := model.GetUserConsumeLogsForAdminExport(userId, startSec, endSec)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}

	username := adminUserExportUsername(userId)
	filename := adminUserExportFilename("consumption-details", userId, username, year, month)
	adminUserExportSetCSVHeaders(c, filename)
	c.Status(http.StatusOK)

	if _, err = c.Writer.Write([]byte{0xEF, 0xBB, 0xBF}); err != nil {
		return
	}
	w := csv.NewWriter(c.Writer)
	writeAdminUserConsumptionDetailsCSV(w, logs, loc)
}

// ExportAdminUserMonthlyBillAndConsumptionDetails ZIP: 同时导出月账单摘要与消费明细。
func ExportAdminUserMonthlyBillAndConsumptionDetails(c *gin.Context) {
	userId, year, month, loc, err := parseAdminUserExportMonthQuery(c)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	startSec, endSec, err := model.AdminUserMonthRangeSeconds(year, month, loc)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}

	username := adminUserExportUsername(userId)
	typeRows, err := model.GetUserLogTypeAggregatesForRange(userId, startSec, endSec)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	modelRows, err := model.GetUserConsumeSummaryByModelForRange(userId, startSec, endSec)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	logs, err := model.GetUserConsumeLogsForAdminExport(userId, startSec, endSec)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}

	zipFilename := strings.TrimSuffix(adminUserExportFilename("monthly-bill-and-consumption-details", userId, username, year, month), ".csv") + ".zip"
	adminUserExportSetZipHeaders(c, zipFilename)
	c.Status(http.StatusOK)

	zipWriter := zip.NewWriter(c.Writer)
	defer zipWriter.Close()

	monthlyFilename := adminUserExportFilename("monthly-bill", userId, username, year, month)
	monthlyFile, err := zipWriter.Create(monthlyFilename)
	if err != nil {
		return
	}
	_, _ = monthlyFile.Write([]byte{0xEF, 0xBB, 0xBF})
	monthlyCSV := csv.NewWriter(monthlyFile)
	writeAdminUserMonthlyBillCSV(monthlyCSV, userId, username, year, month, loc, startSec, endSec, typeRows, modelRows)

	detailsFilename := adminUserExportFilename("consumption-details", userId, username, year, month)
	detailsFile, err := zipWriter.Create(detailsFilename)
	if err != nil {
		return
	}
	_, _ = detailsFile.Write([]byte{0xEF, 0xBB, 0xBF})
	detailsCSV := csv.NewWriter(detailsFile)
	writeAdminUserConsumptionDetailsCSV(detailsCSV, logs, loc)
}
