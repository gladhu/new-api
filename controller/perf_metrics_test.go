package controller

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	perfmetrics "github.com/QuantumNous/new-api/pkg/perf_metrics"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type perfSummaryTestResponse struct {
	Success bool                         `json:"success"`
	Data    perfmetrics.SummaryAllResult `json:"data"`
}

type perfSeriesTestResponse struct {
	Success bool                    `json:"success"`
	Data    perfmetrics.QueryResult `json:"data"`
}

func setupPerfMetricsControllerTestDB(t *testing.T) {
	t.Helper()
	db := setupModelListControllerTestDB(t)
	require.NoError(t, db.AutoMigrate(&model.PerfMetric{}))
	nowBucket := (time.Now().Unix() / 3600) * 3600
	require.NoError(t, model.DB.Create(&model.PerfMetric{
		ModelName:      "gpt-4o",
		Group:          "default",
		BucketTs:       nowBucket,
		RequestCount:   20,
		SuccessCount:   19,
		TotalLatencyMs: 6000,
		TtftSumMs:      2000,
		TtftCount:      20,
		OutputTokens:   400,
		GenerationMs:   8000,
	}).Error)
}

func TestGetPerfMetricsSummaryRedactsCountsByRole(t *testing.T) {
	setupPerfMetricsControllerTestDB(t)

	t.Run("non-admin omits request and success counts", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(recorder)
		ctx.Set("role", common.RoleCommonUser)
		ctx.Request = httptest.NewRequest(http.MethodGet, "/api/perf-metrics/summary?hours=24", nil)

		GetPerfMetricsSummary(ctx)

		require.Equal(t, http.StatusOK, recorder.Code)
		body := recorder.Body.String()
		require.NotContains(t, body, "request_count")
		require.NotContains(t, body, "success_count")

		var payload perfSummaryTestResponse
		require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &payload))
		require.True(t, payload.Success)
		require.Len(t, payload.Data.Models, 1)
		require.Equal(t, int64(0), payload.Data.Models[0].RequestCount)
		require.Equal(t, int64(0), payload.Data.Models[0].SuccessCount)
	})

	t.Run("admin receives request and success counts", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(recorder)
		ctx.Set("role", common.RoleAdminUser)
		ctx.Request = httptest.NewRequest(http.MethodGet, "/api/perf-metrics/summary?hours=24", nil)

		GetPerfMetricsSummary(ctx)

		require.Equal(t, http.StatusOK, recorder.Code)
		var payload perfSummaryTestResponse
		require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &payload))
		require.True(t, payload.Success)
		require.Len(t, payload.Data.Models, 1)
		require.Equal(t, int64(20), payload.Data.Models[0].RequestCount)
		require.Equal(t, int64(19), payload.Data.Models[0].SuccessCount)
	})
}

func TestGetPerfMetricsRedactsSeriesCountsByRole(t *testing.T) {
	setupPerfMetricsControllerTestDB(t)

	t.Run("non-admin omits series counts", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(recorder)
		ctx.Set("role", common.RoleCommonUser)
		ctx.Request = httptest.NewRequest(http.MethodGet, "/api/perf-metrics?model=gpt-4o&hours=24", nil)

		GetPerfMetrics(ctx)

		require.Equal(t, http.StatusOK, recorder.Code)
		body := recorder.Body.String()
		require.NotContains(t, body, "request_count")
		require.NotContains(t, body, "success_count")
	})

	t.Run("admin receives series counts", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(recorder)
		ctx.Set("role", common.RoleAdminUser)
		ctx.Request = httptest.NewRequest(http.MethodGet, "/api/perf-metrics?model=gpt-4o&hours=24", nil)

		GetPerfMetrics(ctx)

		require.Equal(t, http.StatusOK, recorder.Code)
		var payload perfSeriesTestResponse
		require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &payload))
		require.True(t, payload.Success)
		require.Len(t, payload.Data.Groups, 1)
		require.Len(t, payload.Data.Groups[0].Series, 1)
		require.Equal(t, int64(20), payload.Data.Groups[0].Series[0].RequestCount)
		require.Equal(t, int64(19), payload.Data.Groups[0].Series[0].SuccessCount)
	})
}
