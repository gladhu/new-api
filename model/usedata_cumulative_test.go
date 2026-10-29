package model

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetCumulativeQuotaDataSumsHoursAndKeepsModelList(t *testing.T) {
	truncateTables(t)

	require.NoError(t, DB.Create(&QuotaData{
		UserID: 1, Username: "alice", ModelName: "gpt-a", ChannelID: 1, CreatedAt: 1500, Quota: 100,
	}).Error)
	require.NoError(t, DB.Create(&QuotaData{
		UserID: 1, Username: "alice", ModelName: "gpt-a", ChannelID: 2, CreatedAt: 1500, Quota: 25,
	}).Error)
	require.NoError(t, DB.Create(&QuotaData{
		UserID: 1, Username: "alice", ModelName: "gpt-b", CreatedAt: 1600, Quota: 40,
	}).Error)
	require.NoError(t, DB.Create(&QuotaData{
		UserID: 1, Username: "alice", ModelName: "", CreatedAt: 1700, Quota: 9,
	}).Error)
	require.NoError(t, DB.Create(&QuotaData{
		UserID: 2, Username: "bob", ModelName: "gpt-a", CreatedAt: 1500, Quota: 70,
	}).Error)
	require.NoError(t, DB.Create(&QuotaData{
		UserID: 1, Username: "alice", ModelName: "gpt-a", CreatedAt: 5000, Quota: 999,
	}).Error)

	all, err := GetCumulativeQuotaData(1000, 2000, 0, "", "")
	require.NoError(t, err)
	require.Equal(t, []string{"gpt-a", "gpt-b"}, all.Models)
	require.Equal(t, []HourlyQuotaPoint{
		{CreatedAt: 1500, Quota: 195},
		{CreatedAt: 1600, Quota: 40},
		{CreatedAt: 1700, Quota: 9},
	}, all.Points)

	filtered, err := GetCumulativeQuotaData(1000, 2000, 0, "", "gpt-a")
	require.NoError(t, err)
	require.Equal(t, []string{"gpt-a", "gpt-b"}, filtered.Models)
	require.Equal(t, []HourlyQuotaPoint{{CreatedAt: 1500, Quota: 195}}, filtered.Points)

	alice, err := GetCumulativeQuotaData(1000, 2000, 1, "", "")
	require.NoError(t, err)
	require.Equal(t, []string{"gpt-a", "gpt-b"}, alice.Models)
	require.Equal(t, []HourlyQuotaPoint{
		{CreatedAt: 1500, Quota: 125},
		{CreatedAt: 1600, Quota: 40},
		{CreatedAt: 1700, Quota: 9},
	}, alice.Points)

	bob, err := GetCumulativeQuotaData(1000, 2000, 0, "bob", "")
	require.NoError(t, err)
	require.Equal(t, []string{"gpt-a"}, bob.Models)
	require.Equal(t, []HourlyQuotaPoint{{CreatedAt: 1500, Quota: 70}}, bob.Points)
}
