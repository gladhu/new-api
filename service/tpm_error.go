package service

import (
	"net/http"
	"strings"

	"github.com/QuantumNous/new-api/relaykit/types"
)

// IsTPMInsufficientError reports whether an upstream error is a per-minute
// token-rate rejection (TPM).
//
// A 429, rate_limit_exceeded, or no_capacity code alone is not enough because
// request-rate limits (RPM), daily token limits (TPD), billing quota exhaustion
// (insufficient_quota), and Azure peak-load usage-size errors are distinct from
// TPM. Conversely, 400 validation / context-length errors that happen to contain
// "too many tokens" must not be misclassified as TPM.
func IsTPMInsufficientError(err *types.NewAPIError) bool {
	if err == nil || err.StatusCode == http.StatusBadRequest {
		return false
	}

	code := strings.ToLower(strings.TrimSpace(string(err.GetErrorCode())))
	var relayErrType string
	switch relayErr := err.RelayError.(type) {
	case types.OpenAIError:
		relayErrType = strings.ToLower(strings.TrimSpace(relayErr.Type))
	case types.ClaudeError:
		relayErrType = strings.ToLower(strings.TrimSpace(relayErr.Type))
	}

	for _, tag := range []string{code, relayErrType} {
		switch tag {
		case "context_length_exceeded",
			"string_above_max_length",
			"invalid_request_error",
			"validation_error",
			"insufficient_quota":
			return false
		case "token_rate_limit_exceeded",
			"tokens_rate_limit_exceeded":
			return true
		}
	}

	msg := strings.ToLower(err.Error())
	if msg == "" {
		return false
	}

	if strings.Contains(msg, "token rate limit") ||
		strings.Contains(msg, "tokens per min") ||
		strings.Contains(msg, "token limit per min") ||
		strings.Contains(msg, "tokensperminute") ||
		strings.Contains(msg, "(tpm)") {
		return !strings.Contains(msg, "per day") && !strings.Contains(msg, "(tpd)")
	}

	if strings.Contains(msg, "too many tokens") {
		if strings.Contains(msg, "context") ||
			strings.Contains(msg, "max_tokens") ||
			strings.Contains(msg, "per day") ||
			strings.Contains(msg, "(tpd)") {
			return false
		}
		return err.StatusCode == http.StatusTooManyRequests ||
			code == "rate_limit_exceeded" ||
			relayErrType == "rate_limit_exceeded" ||
			relayErrType == "throttlingexception" ||
			code == string(types.ErrorCodeAwsInvokeError) ||
			strings.Contains(msg, "throttling") ||
			strings.Contains(msg, "wait before trying again")
	}

	return false
}
