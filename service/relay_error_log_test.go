package service

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/pkg/billingexpr"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/QuantumNous/new-api/relaykit/types"
	hosttypes "github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func resetTokenQuotaErrorLogGateForTest() {
	tokenQuotaErrorLogGate.mu.Lock()
	tokenQuotaErrorLogGate.seen = make(map[int]tokenQuotaErrorLogEntry)
	tokenQuotaErrorLogGate.mu.Unlock()
}

func withErrorLogEnabled(t *testing.T, enabled bool) {
	t.Helper()
	previous := constant.ErrorLogEnabled
	constant.ErrorLogEnabled = enabled
	t.Cleanup(func() {
		constant.ErrorLogEnabled = previous
	})
}

func newRelayErrorLogContext(path string) *gin.Context {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, path, nil)
	return c
}

func TestRecordRelayErrorLogWritesUsageError(t *testing.T) {
	truncate(t)
	withErrorLogEnabled(t, true)

	c := newRelayErrorLogContext("/v1/chat/completions")
	c.Set("id", 11)
	c.Set("username", "alice")
	c.Set("token_id", 22)
	c.Set("token_name", "my-key")
	c.Set("original_model", "gpt-4")
	c.Set("group", "default")

	apiErr := types.NewErrorWithStatusCode(
		fmt.Errorf("token quota is not enough, token remain quota: $0.000000, need quota: $0.002000"),
		types.ErrorCodePreConsumeTokenQuotaFailed,
		http.StatusForbidden,
		types.ErrOptionWithSkipRetry(),
		types.ErrOptionWithNoRecordErrorLog(),
	)
	RecordRelayErrorLog(c, apiErr)

	log := getLastLog(t)
	require.NotNil(t, log)
	require.Equal(t, model.LogTypeError, log.Type)
	require.Equal(t, 11, log.UserId)
	require.Equal(t, 22, log.TokenId)
	require.Equal(t, "my-key", log.TokenName)
	require.Equal(t, "gpt-4", log.ModelName)
	require.Equal(t, "alice", log.Username)
	require.Contains(t, log.Content, "token quota is not enough")
	require.Contains(t, log.Other, "pre_consume_token_quota_failed")
}

func TestRecordRelayErrorLogSkipsWhenDisabled(t *testing.T) {
	truncate(t)
	withErrorLogEnabled(t, false)

	c := newRelayErrorLogContext("/v1/chat/completions")
	c.Set("id", 11)
	c.Set("token_name", "my-key")

	RecordRelayErrorLog(c, types.NewErrorWithStatusCode(
		fmt.Errorf("token quota is not enough"),
		types.ErrorCodePreConsumeTokenQuotaFailed,
		http.StatusForbidden,
	))

	require.Zero(t, countLogs(t))
}

func TestPreConsumeBillingRecordsTokenQuotaErrorLog(t *testing.T) {
	truncate(t)
	resetTokenQuotaErrorLogGateForTest()
	withErrorLogEnabled(t, false)

	const userID, tokenID, remainQuota, needQuota = 31, 41, 10, 1000
	seedUser(t, userID, 100000)
	seedToken(t, tokenID, userID, "token-quota-log-key", remainQuota)

	c := newRelayErrorLogContext("/v1/chat/completions")
	c.Set("id", userID)
	c.Set("token_id", tokenID)
	c.Set("token_quota", remainQuota)
	c.Set("token_name", "test_token")
	c.Set("original_model", "gpt-4")

	info := &relaycommon.RelayInfo{
		UserId:          userID,
		TokenId:         tokenID,
		TokenKey:        "token-quota-log-key",
		OriginModelName: "gpt-4",
	}

	apiErr := PreConsumeBilling(c, needQuota, info)
	require.NotNil(t, apiErr)
	require.Equal(t, types.ErrorCodePreConsumeTokenQuotaFailed, apiErr.GetErrorCode())
	require.False(t, types.IsRecordErrorLog(apiErr), "channel error path must not double-write this log")

	log := getLastLog(t)
	require.NotNil(t, log)
	require.Equal(t, model.LogTypeError, log.Type)
	require.Equal(t, userID, log.UserId)
	require.Equal(t, tokenID, log.TokenId)
	require.Equal(t, "test_token", log.TokenName)
	require.Contains(t, log.Content, "token quota is not enough")
}

func TestPreConsumeBillingDoesNotRecordTokenQuotaErrorWhenUserHasNoQuota(t *testing.T) {
	truncate(t)
	resetTokenQuotaErrorLogGateForTest()
	withErrorLogEnabled(t, true)

	const userID, tokenID, remainQuota, needQuota = 32, 42, 10, 1000
	seedUser(t, userID, 0)
	seedToken(t, tokenID, userID, "token-no-user-quota", remainQuota)

	c := newRelayErrorLogContext("/v1/chat/completions")
	c.Set("id", userID)
	c.Set("token_id", tokenID)
	c.Set("token_name", "test_token")
	c.Set("original_model", "gpt-4")

	info := &relaycommon.RelayInfo{
		UserId:          userID,
		TokenId:         tokenID,
		TokenKey:        "token-no-user-quota",
		OriginModelName: "gpt-4",
		UserSetting: dto.UserSetting{
			BillingPreference: "subscription_only",
		},
	}

	apiErr := PreConsumeBilling(c, needQuota, info)
	require.NotNil(t, apiErr)
	require.Equal(t, types.ErrorCodePreConsumeTokenQuotaFailed, apiErr.GetErrorCode())
	require.Zero(t, countLogs(t))
}

func TestRecordTokenQuotaErrorLogDedupsSameTokenUntilRemainIncreases(t *testing.T) {
	truncate(t)
	resetTokenQuotaErrorLogGateForTest()

	const userID, tokenID, remainQuota = 33, 43, 50
	seedUser(t, userID, 100000)

	apiErr := types.NewErrorWithStatusCode(
		fmt.Errorf("token quota is not enough, token remain quota: $0.000100, need quota: $0.000644"),
		types.ErrorCodePreConsumeTokenQuotaFailed,
		http.StatusForbidden,
		types.ErrOptionWithSkipRetry(),
	)

	c := newRelayErrorLogContext("/v1/responses")
	c.Set("id", userID)
	c.Set("token_id", tokenID)
	c.Set("token_name", "test_token")
	c.Set("original_model", "gpt-5.6-sol")

	RecordTokenQuotaErrorLog(c, apiErr, userID, remainQuota)
	require.Equal(t, int64(1), countLogs(t))

	RecordTokenQuotaErrorLog(c, apiErr, userID, remainQuota)
	require.Equal(t, int64(1), countLogs(t), "identical retries must not write another error log")

	RecordTokenQuotaErrorLog(c, apiErr, userID, remainQuota-1)
	require.Equal(t, int64(1), countLogs(t), "further depletion must not write another error log")

	RecordTokenQuotaErrorLog(c, apiErr, userID, remainQuota+100)
	require.Equal(t, int64(2), countLogs(t), "a topped-up key running out again should write a new error log")
}

func TestPrepareTieredBillingReserveRecordsTokenQuotaErrorLog(t *testing.T) {
	truncate(t)
	resetTokenQuotaErrorLogGateForTest()

	const userID, tokenID, remainQuota, firstPreConsume = 34, 44, 80, 50
	seedUser(t, userID, 500000)
	seedToken(t, tokenID, userID, "token-reserve-log-key", remainQuota)

	c := newRelayErrorLogContext("/v1/chat/completions")
	c.Set("id", userID)
	c.Set("token_id", tokenID)
	c.Set("token_quota", remainQuota)
	c.Set("token_name", "test_token")
	c.Set("original_model", "gpt-4")

	info := &relaycommon.RelayInfo{
		UserId:          userID,
		TokenId:         tokenID,
		TokenKey:        "token-reserve-log-key",
		OriginModelName: "gpt-4",
		ForcePreConsume: true,
		UserSetting: dto.UserSetting{
			BillingPreference: "wallet_only",
		},
	}

	require.Nil(t, PreConsumeBilling(c, firstPreConsume, info))
	require.NotNil(t, info.Billing)
	require.Zero(t, countLogs(t))

	info.TieredBillingSnapshot = &billingexpr.BillingSnapshot{
		BillingMode:               "tiered_expr",
		ExprString:                `tier("base", p)`,
		ExprHash:                  billingexpr.ExprHashString(`tier("base", p)`),
		GroupRatio:                0.1,
		EstimatedQuotaBeforeGroup: 1000,
		EstimatedQuotaAfterGroup:  firstPreConsume,
		QuotaPerUnit:              500000,
	}
	info.PriceData.GroupRatioInfo = hosttypes.GroupRatioInfo{GroupRatio: 1}

	apiErr := PrepareTieredBillingForSelectedGroup(c, info)
	require.NotNil(t, apiErr)
	require.Equal(t, types.ErrorCodePreConsumeTokenQuotaFailed, apiErr.GetErrorCode())

	log := getLastLog(t)
	require.NotNil(t, log)
	require.Equal(t, model.LogTypeError, log.Type)
	require.Equal(t, userID, log.UserId)
	require.Equal(t, tokenID, log.TokenId)
	require.Contains(t, log.Content, "token quota is not enough")
}
