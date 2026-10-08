package model

import (
	"errors"
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/QuantumNous/new-api/common"

	"gorm.io/gorm"
)

const (
	quotaDataReconcileLockType = "quota_data_reconcile"
	quotaDataReconcileRecent   = 6 * time.Hour
	quotaDataReconcileWide     = 45 * 24 * time.Hour
	quotaDataReconcileLockFor  = 15 * time.Minute
)

var errQuotaDataReconcileBusy = errors.New("quota data reconcile busy")

type quotaRollupKey struct {
	UserID    int
	Username  string
	ModelName string
	CreatedAt int64
	UseGroup  string
	TokenID   int
	ChannelID int
}

type quotaDataRollup struct {
	UserID    int    `gorm:"column:user_id"`
	Username  string `gorm:"column:username"`
	ModelName string `gorm:"column:model_name"`
	CreatedAt int64  `gorm:"column:created_at"`
	UseGroup  string `gorm:"column:use_group"`
	TokenID   int    `gorm:"column:token_id"`
	ChannelID int    `gorm:"column:channel_id"`
	Count     int64  `gorm:"column:cnt"`
	Quota     int64  `gorm:"column:quota"`
	TokenUsed int64  `gorm:"column:token_used"`
}

// ReconcileQuotaDataFromLogs makes closed-hour dashboard totals match consume
// logs. The open hour is left alone because it may still be sitting in the
// memory cache. Hours for keys that are still cached are skipped so a later
// flush cannot add them twice.
//
// Another node already reconciling returns errQuotaDataReconcileBusy.
func ReconcileQuotaDataFromLogs(start, end int64) error {
	start -= start % 3600
	end -= end % 3600
	if start >= end {
		return nil
	}
	now := time.Now().Unix()
	runner := quotaReconcileRunner()
	acquired, _, err := acquireSystemTaskLock(
		quotaDataReconcileLockType,
		quotaDataReconcileLockType,
		runner,
		now,
		now+int64(quotaDataReconcileLockFor/time.Second),
	)
	if err != nil {
		return err
	}
	if !acquired {
		return errQuotaDataReconcileBusy
	}
	defer func() {
		if releaseErr := ReleaseSystemTaskLock(quotaDataReconcileLockType, runner); releaseErr != nil {
			common.SysError("release quota reconcile lock: " + releaseErr.Error())
		}
	}()
	return reconcileQuotaDataFromLogs(start, end)
}

func quotaReconcileRunner() string {
	if common.NodeName != "" {
		return common.NodeName
	}
	return "quota-reconcile"
}

func reconcileQuotaDataFromLogs(start, end int64) error {
	logRows, err := loadConsumeLogRollups(start, end)
	if err != nil {
		return err
	}
	dashRows, err := loadQuotaDataRollups(start, end)
	if err != nil {
		return err
	}
	logs := indexQuotaRollups(logRows)
	dashes := indexQuotaRollups(dashRows)
	cached := cachedQuotaRollupKeys()

	adjusted := 0
	var quotaDelta int64
	keys := make(map[quotaRollupKey]struct{}, len(logs)+len(dashes))
	for key := range logs {
		keys[key] = struct{}{}
	}
	for key := range dashes {
		keys[key] = struct{}{}
	}
	for key := range keys {
		if _, pending := cached[key]; pending {
			continue
		}
		dCount := logs[key].Count - dashes[key].Count
		dQuota := logs[key].Quota - dashes[key].Quota
		dTokens := logs[key].TokenUsed - dashes[key].TokenUsed
		if dCount == 0 && dQuota == 0 && dTokens == 0 {
			continue
		}
		count, countOK := fitQuotaColumn(dCount)
		quota, quotaOK := fitQuotaColumn(dQuota)
		tokenUsed, tokenOK := fitQuotaColumn(dTokens)
		if !countOK || !quotaOK || !tokenOK {
			common.SysError(fmt.Sprintf(
				"quota reconcile skipped %s at %d: delta does not fit the quota_data columns",
				key.ModelName, key.CreatedAt,
			))
			continue
		}
		if err = applyQuotaDataDelta(key, count, quota, tokenUsed); err != nil {
			return err
		}
		adjusted++
		quotaDelta += dQuota
	}
	if adjusted > 0 {
		common.SysLog(fmt.Sprintf("quota data reconcile adjusted %d groups, quota delta %d", adjusted, quotaDelta))
	}
	return nil
}

func loadConsumeLogRollups(start, end int64) ([]quotaDataRollup, error) {
	hourExpr := "(created_at - (created_at % 3600))"
	var rows []quotaDataRollup
	err := LOG_DB.Table("logs").
		Select(
			"user_id, username, model_name, "+hourExpr+" as created_at, "+logGroupCol+" as use_group, token_id, channel_id, "+
				"COUNT(*) as cnt, COALESCE(SUM(quota), 0) as quota, COALESCE(SUM(prompt_tokens), 0) + COALESCE(SUM(completion_tokens), 0) as token_used",
		).
		Where("type = ? AND created_at >= ? AND created_at < ?", LogTypeConsume, start, end).
		Group("user_id, username, model_name, " + hourExpr + ", " + logGroupCol + ", token_id, channel_id").
		Find(&rows).Error
	return rows, err
}

func loadQuotaDataRollups(start, end int64) ([]quotaDataRollup, error) {
	var rows []quotaDataRollup
	err := DB.Table("quota_data").
		Select(
			"user_id, username, model_name, created_at, use_group, token_id, channel_id, "+
				"COALESCE(SUM(count), 0) as cnt, COALESCE(SUM(quota), 0) as quota, COALESCE(SUM(token_used), 0) as token_used",
		).
		Where("created_at >= ? AND created_at < ?", start, end).
		Group("user_id, username, model_name, created_at, use_group, token_id, channel_id").
		Find(&rows).Error
	return rows, err
}

func indexQuotaRollups(rows []quotaDataRollup) map[quotaRollupKey]quotaDataRollup {
	indexed := make(map[quotaRollupKey]quotaDataRollup, len(rows))
	for _, row := range rows {
		key := quotaRollupKey{
			UserID:    row.UserID,
			Username:  row.Username,
			ModelName: row.ModelName,
			CreatedAt: row.CreatedAt,
			UseGroup:  row.UseGroup,
			TokenID:   row.TokenID,
			ChannelID: row.ChannelID,
		}
		current := indexed[key]
		current.Count += row.Count
		current.Quota += row.Quota
		current.TokenUsed += row.TokenUsed
		indexed[key] = current
	}
	return indexed
}

func cachedQuotaRollupKeys() map[quotaRollupKey]struct{} {
	CacheQuotaDataLock.Lock()
	defer CacheQuotaDataLock.Unlock()
	keys := make(map[quotaRollupKey]struct{}, len(CacheQuotaData))
	for _, row := range CacheQuotaData {
		keys[quotaRollupKey{
			UserID:    row.UserID,
			Username:  row.Username,
			ModelName: row.ModelName,
			CreatedAt: row.CreatedAt,
			UseGroup:  row.UseGroup,
			TokenID:   row.TokenID,
			ChannelID: row.ChannelID,
		}] = struct{}{}
	}
	return keys
}

func applyQuotaDataDelta(key quotaRollupKey, count int, quota int, tokenUsed int) error {
	if count > 0 || quota > 0 || tokenUsed > 0 {
		if err := addQuotaDataDelta(key, positive(count), positive(quota), positive(tokenUsed)); err != nil {
			return err
		}
	}
	if count < 0 || quota < 0 || tokenUsed < 0 {
		return subtractQuotaDataDelta(key, positive(-count), positive(-quota), positive(-tokenUsed))
	}
	return nil
}

func addQuotaDataDelta(key quotaRollupKey, count int, quota int, tokenUsed int) error {
	var existing QuotaData
	err := DB.Where(
		"user_id = ? AND username = ? AND model_name = ? AND created_at = ? AND use_group = ? AND token_id = ? AND channel_id = ? AND node_name = ?",
		key.UserID, key.Username, key.ModelName, key.CreatedAt, key.UseGroup, key.TokenID, key.ChannelID, "",
	).First(&existing).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return DB.Create(&QuotaData{
			UserID:    key.UserID,
			Username:  key.Username,
			ModelName: key.ModelName,
			CreatedAt: key.CreatedAt,
			UseGroup:  key.UseGroup,
			TokenID:   key.TokenID,
			ChannelID: key.ChannelID,
			Count:     count,
			Quota:     quota,
			TokenUsed: tokenUsed,
		}).Error
	}
	if err != nil {
		return err
	}
	if existing.Count > math.MaxInt32-count || existing.Quota > math.MaxInt32-quota || existing.TokenUsed > math.MaxInt32-tokenUsed {
		return fmt.Errorf("quota reconcile overflow for %s at %d", key.ModelName, key.CreatedAt)
	}
	return increaseQuotaData(existing.Id, count, quota, tokenUsed)
}

func subtractQuotaDataDelta(key quotaRollupKey, count int, quota int, tokenUsed int) error {
	var rows []QuotaData
	err := DB.Where(
		"user_id = ? AND username = ? AND model_name = ? AND created_at = ? AND use_group = ? AND token_id = ? AND channel_id = ?",
		key.UserID, key.Username, key.ModelName, key.CreatedAt, key.UseGroup, key.TokenID, key.ChannelID,
	).Find(&rows).Error
	if err != nil {
		return err
	}
	sort.SliceStable(rows, func(i, j int) bool {
		if (rows[i].NodeName == "") != (rows[j].NodeName == "") {
			return rows[i].NodeName == ""
		}
		return rows[i].Id < rows[j].Id
	})
	for _, row := range rows {
		if count == 0 && quota == 0 && tokenUsed == 0 {
			break
		}
		takeCount := minAvailable(row.Count, count)
		takeQuota := minAvailable(row.Quota, quota)
		takeTokens := minAvailable(row.TokenUsed, tokenUsed)
		if takeCount == 0 && takeQuota == 0 && takeTokens == 0 {
			continue
		}
		result := DB.Model(&QuotaData{}).Where("id = ?", row.Id).Updates(map[string]interface{}{
			"count":      gorm.Expr("count - ?", takeCount),
			"quota":      gorm.Expr("quota - ?", takeQuota),
			"token_used": gorm.Expr("token_used - ?", takeTokens),
		})
		if result.Error != nil {
			return result.Error
		}
		count -= takeCount
		quota -= takeQuota
		tokenUsed -= takeTokens
	}
	if count != 0 || quota != 0 || tokenUsed != 0 {
		common.SysError(fmt.Sprintf(
			"quota reconcile could not fully reduce %s at %d: count %d quota %d tokens %d left",
			key.ModelName, key.CreatedAt, count, quota, tokenUsed,
		))
	}
	return nil
}

func fitQuotaColumn(value int64) (int, bool) {
	if value > math.MaxInt32 || value < math.MinInt32 {
		return 0, false
	}
	return int(value), true
}

func positive(value int) int {
	if value < 0 {
		return 0
	}
	return value
}

func minAvailable(have int, need int) int {
	if have < 0 {
		have = 0
	}
	if need < have {
		return need
	}
	return have
}
