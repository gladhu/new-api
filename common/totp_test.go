package common

import (
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/pquerna/otp/totp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateQRCodeDataEncodesIssuerAndAccountName(t *testing.T) {
	t.Setenv("SYSTEM_NAME", "New API")

	qr := GenerateQRCodeData("JBSWY3DPEHPK3PXP", "root (2)")

	parsed, err := url.Parse(qr)
	require.NoError(t, err)
	assert.Equal(t, "otpauth", parsed.Scheme)
	assert.Equal(t, "totp", parsed.Host)
	assert.Contains(t, parsed.Path, "New%20API")
	assert.Equal(t, "JBSWY3DPEHPK3PXP", parsed.Query().Get("secret"))
	assert.Equal(t, "New API", parsed.Query().Get("issuer"))
}

func TestGenerateTOTPSecretURLMatchesValidation(t *testing.T) {
	key, err := GenerateTOTPSecret("root")
	require.NoError(t, err)

	code, err := totp.GenerateCode(key.Secret(), time.Now())
	require.NoError(t, err)
	assert.True(t, ValidateTOTPCode(key.Secret(), code))
	assert.True(t, strings.HasPrefix(key.URL(), "otpauth://totp/"))
}
