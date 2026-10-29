package controller

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type cumulativeQuotaResponse struct {
	Success bool                  `json:"success"`
	Message string                `json:"message"`
	Data    model.CumulativeQuota `json:"data"`
}

func setupCumulativeControllerTestDB(t *testing.T) {
	t.Helper()
	db := setupModelListControllerTestDB(t)
	require.NoError(t, db.AutoMigrate(&model.QuotaData{}))
	require.NoError(t, model.DB.Create(&model.QuotaData{
		UserID: 1, Username: "alice", ModelName: "gpt-a", CreatedAt: 1500, Quota: 100,
	}).Error)
	require.NoError(t, model.DB.Create(&model.QuotaData{
		UserID: 2, Username: "bob", ModelName: "gpt-b", CreatedAt: 1600, Quota: 70,
	}).Error)
}

func decodeCumulativeQuotaBody(t *testing.T, recorder *httptest.ResponseRecorder) cumulativeQuotaResponse {
	t.Helper()
	require.Equal(t, http.StatusOK, recorder.Code)
	var payload cumulativeQuotaResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &payload))
	return payload
}

func TestGetCumulativeQuotaRejectsASpanBeyondOneMonth(t *testing.T) {
	setupCumulativeControllerTestDB(t)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/data/cumulative?start_timestamp=1&end_timestamp=2764802", nil)

	GetCumulativeQuota(ctx)

	payload := decodeCumulativeQuotaBody(t, recorder)
	require.False(t, payload.Success)
	require.Equal(t, "时间跨度不能超过 1 个月", payload.Message)
}

func TestGetCumulativeQuotaFiltersByUsername(t *testing.T) {
	setupCumulativeControllerTestDB(t)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/data/cumulative?start_timestamp=1000&end_timestamp=2000&username=bob", nil)

	GetCumulativeQuota(ctx)

	payload := decodeCumulativeQuotaBody(t, recorder)
	require.True(t, payload.Success, payload.Message)
	require.Equal(t, []string{"gpt-b"}, payload.Data.Models)
	require.Equal(t, []model.HourlyQuotaPoint{{CreatedAt: 1600, Quota: 70}}, payload.Data.Points)
}

func TestGetUserCumulativeQuotaIgnoresUsernameAndUsesTheSessionUser(t *testing.T) {
	setupCumulativeControllerTestDB(t)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Set("id", 1)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/data/cumulative/self?start_timestamp=1000&end_timestamp=2000&username=bob", nil)

	GetUserCumulativeQuota(ctx)

	payload := decodeCumulativeQuotaBody(t, recorder)
	require.True(t, payload.Success, payload.Message)
	require.Equal(t, []string{"gpt-a"}, payload.Data.Models)
	require.Equal(t, []model.HourlyQuotaPoint{{CreatedAt: 1500, Quota: 100}}, payload.Data.Points)
}
