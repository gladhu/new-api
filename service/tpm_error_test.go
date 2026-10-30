package service

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/QuantumNous/new-api/relaykit/types"
	"github.com/stretchr/testify/require"
)

func TestIsTPMInsufficientError(t *testing.T) {
	tests := []struct {
		name string
		err  *types.NewAPIError
		want bool
	}{
		{
			name: "azure deployment token rate limit",
			err: types.WithOpenAIError(types.OpenAIError{
				Message: "Your requests to gpt-6.1-sol for gpt-6.1-sol-1 in westus3 have exceeded token rate limit.",
				Type:    "openai_error",
				Code:    "rate_limit_exceeded",
			}, http.StatusTooManyRequests),
			want: true,
		},
		{
			name: "responses api token rate limit",
			err: types.WithOpenAIError(types.OpenAIError{
				Message: "Requests to the Responses API have exceeded the token rate limit.",
				Type:    "openai_error",
				Code:    "rate_limit_exceeded",
			}, http.StatusTooManyRequests),
			want: true,
		},
		{
			name: "openai tokens per min tpm",
			err: types.WithOpenAIError(types.OpenAIError{
				Message: "Rate limit reached for gpt-6.1-sol in organization org_x on tokens per min (TPM): Limit 30000, Used 30000, Requested 100.",
				Type:    "tokens",
				Code:    "rate_limit_exceeded",
			}, http.StatusTooManyRequests),
			want: true,
		},
		{
			name: "anthropic input tokens per minute",
			err: types.WithClaudeError(types.ClaudeError{
				Type:    "rate_limit_error",
				Message: "This request would exceed your organization's rate limit of 40,000 input tokens per minute.",
			}, http.StatusTooManyRequests),
			want: true,
		},
		{
			name: "gemini camelcase tokens per minute quota metric",
			err: types.WithOpenAIError(types.OpenAIError{
				Message: "Quota exceeded for quota metric 'Generate Content input token count' and limit 'GenerateContentInputTokensPerMinutePerProject-Region'.",
				Type:    "RESOURCE_EXHAUSTED",
				Code:    429,
			}, http.StatusTooManyRequests),
			want: true,
		},
		{
			name: "structured token_rate_limit_exceeded code",
			err: types.WithOpenAIError(types.OpenAIError{
				Message: "Rate limit exceeded.",
				Type:    "rate_limit_error",
				Code:    "token_rate_limit_exceeded",
			}, http.StatusTooManyRequests),
			want: true,
		},
		{
			name: "bedrock too many tokens on 429",
			err: types.WithOpenAIError(types.OpenAIError{
				Message: "Too many tokens, please wait before trying again.",
				Type:    "ThrottlingException",
			}, http.StatusTooManyRequests),
			want: true,
		},
		{
			name: "bedrock sdk invoke error with throttling too many tokens",
			err: types.NewOpenAIError(
				fmt.Errorf("ConverseStream: operation error Bedrock Runtime: ConverseStream, https response error StatusCode: 429, ThrottlingException: Too many tokens, please wait before trying again."),
				types.ErrorCodeAwsInvokeError,
				http.StatusInternalServerError,
			),
			want: true,
		},
		{
			name: "status code mapped from 429 to 503 still matches token rate limit",
			err: types.WithOpenAIError(types.OpenAIError{
				Message: "Your requests to gpt-6.1-sol for gpt-6.1-sol-1 in westus3 have exceeded token rate limit.",
				Type:    "openai_error",
				Code:    "rate_limit_exceeded",
			}, http.StatusServiceUnavailable),
			want: true,
		},
		{
			name: "400 prompt too many tokens is context length not tpm",
			err: types.WithOpenAIError(types.OpenAIError{
				Message: "Prompt contains too many tokens: 300000 > 272000.",
				Type:    "invalid_request_error",
				Code:    "context_length_exceeded",
			}, http.StatusBadRequest),
			want: false,
		},
		{
			name: "500 upstream error mentioning too many tokens in context is not tpm",
			err: types.NewOpenAIError(
				fmt.Errorf("request has too many tokens for model context window"),
				types.ErrorCodeBadResponseStatusCode,
				http.StatusInternalServerError,
			),
			want: false,
		},
		{
			name: "tokens per day tpd is daily quota not tpm",
			err: types.WithOpenAIError(types.OpenAIError{
				Message: "Rate limit reached on tokens per day (TPD): Limit 1000000, Used 1000000.",
				Type:    "tokens",
				Code:    "rate_limit_exceeded",
			}, http.StatusTooManyRequests),
			want: false,
		},
		{
			name: "insufficient_quota is billing quota not tpm",
			err: types.WithOpenAIError(types.OpenAIError{
				Message: "You exceeded your current quota, please check your plan and billing details.",
				Type:    "insufficient_quota",
				Code:    "insufficient_quota",
			}, http.StatusTooManyRequests),
			want: false,
		},
		{
			name: "azure call rate limit is rpm",
			err: types.WithOpenAIError(types.OpenAIError{
				Message: "Requests to the ChatCompletions_Create Operation under Azure OpenAI API have exceeded call rate limit of your current AIServices S0 pricing tier.",
				Type:    "openai_error",
				Code:    "rate_limit_exceeded",
			}, http.StatusTooManyRequests),
			want: false,
		},
		{
			name: "openai requests per min is rpm",
			err: types.WithOpenAIError(types.OpenAIError{
				Message: "Rate limit reached for gpt-6.1-sol in organization org_x on requests per min (RPM): Limit 100, Used 100, Requested 1.",
				Type:    "requests",
				Code:    "rate_limit_exceeded",
			}, http.StatusTooManyRequests),
			want: false,
		},
		{
			name: "bedrock too many requests is rpm",
			err: types.WithOpenAIError(types.OpenAIError{
				Message: "Too many requests, please wait before trying again.",
				Type:    "ThrottlingException",
			}, http.StatusTooManyRequests),
			want: false,
		},
		{
			name: "azure peak usage size is not tpm",
			err: types.WithOpenAIError(types.OpenAIError{
				Message: "The system is currently experiencing high demand and cannot process your request. Your request exceeds the maximum usage size allowed during peak load. For improved capacity reliability, consider switching to Provisioned Throughput.",
				Type:    "openai_error",
				Code:    "no_capacity",
			}, http.StatusInternalServerError),
			want: false,
		},
		{
			name: "bedrock generic 503",
			err: types.WithOpenAIError(types.OpenAIError{
				Message: "The server had an error while processing your request. Sorry about that!",
				Type:    "openai_error",
				Code:    "service_unavailable",
			}, http.StatusServiceUnavailable),
			want: false,
		},
		{
			name: "rate limit code without token text",
			err: types.WithOpenAIError(types.OpenAIError{
				Message: "Rate limit exceeded.",
				Type:    "openai_error",
				Code:    "rate_limit_exceeded",
			}, http.StatusTooManyRequests),
			want: false,
		},
		{
			name: "nil",
			err:  nil,
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, IsTPMInsufficientError(tt.err))
		})
	}
}

func TestRecordRelayErrorLogMarksTPMInsufficient(t *testing.T) {
	truncate(t)
	withErrorLogEnabled(t, true)

	c := newRelayErrorLogContext("/v1/chat/completions")
	c.Set("id", 26)
	c.Set("username", "Efund_AWS_001")
	c.Set("token_name", "OneAPI-生产环境")
	c.Set("original_model", "gpt-6.1-sol")
	c.Set("group", "mix")
	c.Set("channel_id", 32)
	c.Set("channel_name", "Azure-gpt-feisi-2383-6.1")

	apiErr := types.WithOpenAIError(types.OpenAIError{
		Message: "Your requests to gpt-6.1-sol for gpt-6.1-sol-1 in westus3 have exceeded token rate limit.",
		Type:    "openai_error",
		Code:    "rate_limit_exceeded",
	}, http.StatusTooManyRequests)
	RecordRelayErrorLog(c, apiErr)

	log := getLastLog(t)
	require.NotNil(t, log)
	require.Contains(t, log.Content, "exceeded token rate limit")
	require.Contains(t, log.Other, `"error_reason":"tpm_insufficient"`)

	c2 := newRelayErrorLogContext("/v1/chat/completions")
	c2.Set("id", 26)
	c2.Set("username", "Efund_AWS_001")
	c2.Set("original_model", "gpt-6.1-sol")
	RecordRelayErrorLog(c2, types.NewOpenAIError(
		fmt.Errorf("The server had an error while processing your request. Sorry about that!"),
		types.ErrorCodeBadResponseStatusCode,
		http.StatusServiceUnavailable,
	))
	logs := countLogs(t)
	require.Equal(t, int64(2), logs)
	latest := getLastLog(t)
	require.NotContains(t, latest.Other, "tpm_insufficient")
}
