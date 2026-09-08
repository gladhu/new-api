package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupTokenAuthQuotaLogTest(t *testing.T) {
	t.Helper()
	previousDB, previousLogDB := model.DB, model.LOG_DB
	previousType := common.MainDatabaseType()
	previousRedis := common.RedisEnabled
	previousErrorLog := constant.ErrorLogEnabled

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.User{}, &model.Token{}, &model.Log{}))

	model.DB = db
	model.LOG_DB = db
	common.SetMainDatabaseType(common.DatabaseTypeSQLite)
	model.InitColumnNames()
	common.RedisEnabled = false
	constant.ErrorLogEnabled = false

	t.Cleanup(func() {
		model.DB = previousDB
		model.LOG_DB = previousLogDB
		common.SetMainDatabaseType(previousType)
		common.RedisEnabled = previousRedis
		constant.ErrorLogEnabled = previousErrorLog
	})
}

func TestTokenAuthRecordsErrorLogWhenKeyQuotaExhausted(t *testing.T) {
	setupTokenAuthQuotaLogTest(t)

	user := &model.User{
		Username: "quota-user",
		Password: "password-placeholder",
		Role:     common.RoleCommonUser,
		Status:   common.UserStatusEnabled,
		Group:    "default",
		Quota:    100000,
	}
	require.NoError(t, model.DB.Create(user).Error)

	token := &model.Token{
		UserId:      user.Id,
		Key:         "quotaexhaustedkey",
		Name:        "user-key",
		Status:      common.TokenStatusExhausted,
		ExpiredTime: -1,
		RemainQuota: 0,
		Group:       "default",
	}
	require.NoError(t, token.Insert())

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	router := gin.New()
	router.Use(TokenAuth())
	router.POST("/v1/chat/completions", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	req.Header.Set("Authorization", "Bearer sk-quotaexhaustedkey")
	router.ServeHTTP(recorder, req)

	require.Equal(t, http.StatusUnauthorized, recorder.Code)

	var body map[string]any
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &body))
	errObj, ok := body["error"].(map[string]any)
	require.True(t, ok)
	message, _ := errObj["message"].(string)
	require.NotEmpty(t, message)
	require.NotContains(t, message, "quota")

	var log model.Log
	require.NoError(t, model.LOG_DB.Order("id desc").First(&log).Error)
	require.Equal(t, model.LogTypeError, log.Type)
	require.Equal(t, user.Id, log.UserId)
	require.Equal(t, token.Id, log.TokenId)
	require.Equal(t, "user-key", log.TokenName)
	require.Contains(t, log.Content, "token quota is not enough")
}

func TestTokenAuthDoesNotRecordErrorLogForUnknownKey(t *testing.T) {
	setupTokenAuthQuotaLogTest(t)

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	router := gin.New()
	router.Use(TokenAuth())
	router.POST("/v1/chat/completions", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	req.Header.Set("Authorization", "Bearer sk-unknownkey")
	router.ServeHTTP(recorder, req)

	require.Equal(t, http.StatusUnauthorized, recorder.Code)

	var count int64
	require.NoError(t, model.LOG_DB.Model(&model.Log{}).Count(&count).Error)
	require.Zero(t, count)
}

func TestTokenAuthDoesNotRecordErrorLogWhenUserHasNoQuota(t *testing.T) {
	setupTokenAuthQuotaLogTest(t)

	user := &model.User{
		Username: "empty-wallet-user",
		Password: "password-placeholder",
		Role:     common.RoleCommonUser,
		Status:   common.UserStatusEnabled,
		Group:    "default",
		Quota:    0,
	}
	require.NoError(t, model.DB.Create(user).Error)

	token := &model.Token{
		UserId:      user.Id,
		Key:         "emptywalletkey",
		Name:        "user-key",
		Status:      common.TokenStatusExhausted,
		ExpiredTime: -1,
		RemainQuota: 0,
		Group:       "default",
	}
	require.NoError(t, token.Insert())

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	router := gin.New()
	router.Use(TokenAuth())
	router.POST("/v1/chat/completions", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	req.Header.Set("Authorization", "Bearer sk-emptywalletkey")
	router.ServeHTTP(recorder, req)

	require.Equal(t, http.StatusUnauthorized, recorder.Code)

	var count int64
	require.NoError(t, model.LOG_DB.Model(&model.Log{}).Count(&count).Error)
	require.Zero(t, count)
}
