package perfmetrics

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/require"
)

func TestBucketPointIncludesCounts(t *testing.T) {
	point := bucketPoint(1_700_000_000, counters{
		requestCount: 10,
		successCount: 8,
	})

	require.Equal(t, int64(10), point.RequestCount)
	require.Equal(t, int64(8), point.SuccessCount)
	require.InDelta(t, 80.0, point.SuccessRate, 0.001)
}

func TestSuccessRateIsRequestWeighted(t *testing.T) {
	require.InDelta(t, 95.0, successRate(counters{
		requestCount: 100,
		successCount: 95,
	}), 0.001)
	require.Equal(t, 0.0, successRate(counters{}))
}

func TestBucketStartAlignsWindowFloor(t *testing.T) {
	endTs := int64(14*3600 + 5*60)
	startTs := bucketStart(endTs - 3600)
	require.Equal(t, int64(13*3600), startTs)
}

func TestRedactedCountsOmittedFromJSON(t *testing.T) {
	summary := ModelSummary{
		ModelName:   "gpt-4o",
		SuccessRate: 98.5,
	}
	raw, err := common.Marshal(summary)
	require.NoError(t, err)
	require.NotContains(t, string(raw), "request_count")
	require.NotContains(t, string(raw), "success_count")
}
