package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/require"
)

func TestValidateUserTokenReturnsExhaustedWhenRemainQuotaUsedUp(t *testing.T) {
	truncateTables(t)

	token := Token{
		UserId:      1,
		Key:         "exhausted-remain-" + common.GetRandomString(8),
		Name:        "exhausted-remain",
		Status:      common.TokenStatusEnabled,
		ExpiredTime: -1,
		RemainQuota: 0,
	}
	require.NoError(t, token.Insert())

	got, err := ValidateUserToken(token.Key)
	require.ErrorIs(t, err, ErrTokenExhausted)
	require.NotNil(t, got)
	require.Equal(t, token.Id, got.Id)
	require.Equal(t, common.TokenStatusExhausted, got.Status)
}

func TestValidateUserTokenReturnsExhaustedWhenStatusIsExhausted(t *testing.T) {
	truncateTables(t)

	token := Token{
		UserId:      1,
		Key:         "exhausted-status-" + common.GetRandomString(8),
		Name:        "exhausted-status",
		Status:      common.TokenStatusExhausted,
		ExpiredTime: -1,
		RemainQuota: 0,
	}
	require.NoError(t, token.Insert())

	got, err := ValidateUserToken(token.Key)
	require.ErrorIs(t, err, ErrTokenExhausted)
	require.NotNil(t, got)
	require.Equal(t, token.Id, got.Id)
}

func TestValidateUserTokenKeepsUnknownKeyAsInvalid(t *testing.T) {
	truncateTables(t)

	got, err := ValidateUserToken("missing-token-key")
	require.ErrorIs(t, err, ErrTokenInvalid)
	require.Nil(t, got)
}
