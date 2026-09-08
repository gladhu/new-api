package service

import (
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/relaykit/types"

	"github.com/gin-gonic/gin"
)

const tokenQuotaErrorLogCooldown = 10 * time.Minute

type tokenQuotaErrorLogEntry struct {
	remain int
	until  time.Time
}

var tokenQuotaErrorLogGate = struct {
	mu   sync.Mutex
	seen map[int]tokenQuotaErrorLogEntry
}{
	seen: make(map[int]tokenQuotaErrorLogEntry),
}

// userHasWalletQuota reports whether the user wallet still has remaining quota.
// Token-key exhaustion is only written to usage logs when the user can still pay.
func userHasWalletQuota(userId int) bool {
	if userId <= 0 {
		return false
	}
	quota, err := model.GetUserQuota(userId, false)
	return err == nil && quota > 0
}

// RecordTokenQuotaErrorLog writes a usage-log error when the user wallet still
// has quota but the token key cannot cover this request. This is a user-facing
// billing rejection and is recorded even if ERROR_LOG_ENABLED is off.
// The same token is only logged again after the cooldown, or sooner if its
// remaining quota increased (the key was topped up and then ran out again).
func RecordTokenQuotaErrorLog(c *gin.Context, apiErr *types.NewAPIError, userId int, remainQuota int) {
	if c == nil || apiErr == nil || !userHasWalletQuota(userId) {
		return
	}
	if !shouldRecordTokenQuotaError(c.GetInt("token_id"), remainQuota) {
		return
	}
	recordRelayErrorLog(c, apiErr)
}

func shouldRecordTokenQuotaError(tokenId int, remainQuota int) bool {
	if tokenId <= 0 {
		return true
	}
	now := time.Now()
	if common.RedisEnabled {
		key := fmt.Sprintf("token_quota_error_log:%d", tokenId)
		if prev, err := common.RedisGet(key); err == nil {
			if prevRemain, parseErr := strconv.Atoi(prev); parseErr == nil && remainQuota <= prevRemain {
				return false
			}
		}
		if err := common.RedisSet(key, strconv.Itoa(remainQuota), tokenQuotaErrorLogCooldown); err == nil {
			return true
		}
	}

	tokenQuotaErrorLogGate.mu.Lock()
	defer tokenQuotaErrorLogGate.mu.Unlock()
	if tokenQuotaErrorLogGate.seen == nil {
		tokenQuotaErrorLogGate.seen = make(map[int]tokenQuotaErrorLogEntry)
	}
	if len(tokenQuotaErrorLogGate.seen) > 4096 {
		for id, entry := range tokenQuotaErrorLogGate.seen {
			if now.After(entry.until) {
				delete(tokenQuotaErrorLogGate.seen, id)
			}
		}
	}
	if entry, ok := tokenQuotaErrorLogGate.seen[tokenId]; ok && now.Before(entry.until) && remainQuota <= entry.remain {
		return false
	}
	tokenQuotaErrorLogGate.seen[tokenId] = tokenQuotaErrorLogEntry{remain: remainQuota, until: now.Add(tokenQuotaErrorLogCooldown)}
	return true
}

// RecordRelayErrorLog writes a usage-log error row when ERROR_LOG_ENABLED is on.
// Callers that must skip logging (for example channel errors marked
// ErrOptionWithNoRecordErrorLog) should check types.IsRecordErrorLog first.
func RecordRelayErrorLog(c *gin.Context, apiErr *types.NewAPIError) {
	if c == nil || apiErr == nil || !constant.ErrorLogEnabled {
		return
	}
	recordRelayErrorLog(c, apiErr)
}

func recordRelayErrorLog(c *gin.Context, apiErr *types.NewAPIError) {
	userId := c.GetInt("id")
	tokenName := c.GetString("token_name")
	modelName := c.GetString("original_model")
	tokenId := c.GetInt("token_id")
	userGroup := c.GetString("group")
	channelId := c.GetInt("channel_id")
	other := make(map[string]interface{})
	if c.Request != nil && c.Request.URL != nil {
		other["request_path"] = c.Request.URL.Path
	}
	other["error_type"] = apiErr.GetErrorType()
	other["error_code"] = apiErr.GetErrorCode()
	other["status_code"] = apiErr.StatusCode
	other["channel_id"] = channelId
	other["channel_name"] = c.GetString("channel_name")
	other["channel_type"] = c.GetInt("channel_type")
	adminInfo := make(map[string]interface{})
	adminInfo["use_channel"] = c.GetStringSlice("use_channel")
	isMultiKey := common.GetContextKeyBool(c, constant.ContextKeyChannelIsMultiKey)
	if isMultiKey {
		adminInfo["is_multi_key"] = true
		adminInfo["multi_key_index"] = common.GetContextKeyInt(c, constant.ContextKeyChannelMultiKeyIndex)
	}
	AppendChannelAffinityAdminInfo(c, adminInfo)
	other["admin_info"] = adminInfo
	startTime := common.GetContextKeyTime(c, constant.ContextKeyRequestStartTime)
	if startTime.IsZero() {
		startTime = time.Now()
	}
	useTimeSeconds := int(time.Since(startTime).Seconds())
	model.RecordErrorLog(c, userId, channelId, modelName, tokenName, apiErr.MaskSensitiveErrorWithStatusCode(), tokenId, useTimeSeconds, common.GetContextKeyBool(c, constant.ContextKeyIsStream), userGroup, other)
}
