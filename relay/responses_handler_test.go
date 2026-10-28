package relay

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/QuantumNous/new-api/relaykit/types"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func TestIsEncryptedReasoningError(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		err        *types.NewAPIError
		want       bool
	}{
		{
			name:       "azure encrypted content could not be verified",
			statusCode: http.StatusBadRequest,
			err: types.WithOpenAIError(types.OpenAIError{
				Message: "The encrypted content for item rs_0cd7fd73d7f57593016adb1d5791988190b5cbb3e119c6c197 could not be verified. Reason: Encrypted content could not be decrypted or parsed.",
				Type:    "invalid_request_error",
			}, http.StatusBadRequest),
			want: true,
		},
		{
			name:       "bedrock invalid encrypted reasoning",
			statusCode: http.StatusBadRequest,
			err: types.WithOpenAIError(types.OpenAIError{
				Message: "invalid encrypted reasoning",
				Type:    "invalid_request_error",
			}, http.StatusBadRequest),
			want: true,
		},
		{
			name:       "bedrock encrypted reasoning created for a different account or provider",
			statusCode: http.StatusBadRequest,
			err: types.WithOpenAIError(types.OpenAIError{
				Message: "encrypted reasoning was created for a different account or provider",
				Type:    "invalid_request_error",
			}, http.StatusBadRequest),
			want: true,
		},
		{
			name:       "openai error code invalid_encrypted_content",
			statusCode: http.StatusBadRequest,
			err: types.WithOpenAIError(types.OpenAIError{
				Message: "Invalid encrypted content",
				Code:    "invalid_encrypted_content",
			}, http.StatusBadRequest),
			want: true,
		},
		{
			name:       "bedrock bare invalid encrypted content without code",
			statusCode: http.StatusBadRequest,
			err: types.WithOpenAIError(types.OpenAIError{
				Message: "invalid encrypted content",
				Type:    "invalid_request_error",
			}, http.StatusBadRequest),
			want: true,
		},
		{
			name:       "bedrock message field invalid encrypted content",
			statusCode: http.StatusBadRequest,
			err: types.NewOpenAIError(
				fmt.Errorf("invalid encrypted content"),
				types.ErrorCodeBadResponseStatusCode,
				http.StatusBadRequest,
			),
			want: true,
		},
		{
			name:       "unrelated 400 error",
			statusCode: http.StatusBadRequest,
			err: types.WithOpenAIError(types.OpenAIError{
				Message: "Invalid parameter: 'temperature' is not supported with this model.",
				Type:    "invalid_request_error",
			}, http.StatusBadRequest),
			want: false,
		},
		{
			name:       "non-400 status code",
			statusCode: http.StatusInternalServerError,
			err: types.WithOpenAIError(types.OpenAIError{
				Message: "invalid encrypted reasoning",
			}, http.StatusInternalServerError),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, isEncryptedReasoningError(tt.statusCode, tt.err))
		})
	}
}

func TestStripEncryptedReasoningFromResponsesBody(t *testing.T) {
	rawBody := []byte(`{
		"model": "gpt-5.4",
		"input": [
			{"type": "message", "role": "user", "content": [{"type": "input_text", "text": "hello"}]},
			{"id": "rs_0cd7fd73d7f57593016adb1d5791988190b5cbb3e119c6c197", "type": "reasoning", "summary": [], "encrypted_content": "gAAAAAB..."},
			{"type": "function_call", "id": "fc_1", "call_id": "call_1", "name": "ls", "arguments": "{}"},
			{"type": "function_call_output", "call_id": "call_1", "output": "ok"}
		],
		"include": ["reasoning.encrypted_content"],
		"store": false
	}`)

	strippedBody, stripped := stripEncryptedReasoningFromResponsesBody(rawBody)
	require.True(t, stripped)

	inputArr := gjson.GetBytes(strippedBody, "input").Array()
	require.Len(t, inputArr, 3)
	assert.Equal(t, "message", inputArr[0].Get("type").String())
	assert.Equal(t, "function_call", inputArr[1].Get("type").String())
	assert.Equal(t, "function_call_output", inputArr[2].Get("type").String())
	assert.Equal(t, "reasoning.encrypted_content", gjson.GetBytes(strippedBody, "include.0").String())

	// Calling a second time on already stripped body returns false.
	_, strippedAgain := stripEncryptedReasoningFromResponsesBody(strippedBody)
	assert.False(t, strippedAgain)
}

func TestStripEncryptedReasoningRemovesNestedAndCompactionCiphertext(t *testing.T) {
	rawBody := []byte(`{
		"model": "gpt-5.6-sol",
		"input": [
			{"type": "message", "role": "user", "content": [{"type": "input_text", "text": "hello"}]},
			{"type": "compaction", "id": "cmp_1", "encrypted_content": "gAAAAAB-compaction"},
			{
				"type": "message",
				"role": "assistant",
				"content": [
					{"type": "output_text", "text": "answer", "encrypted_content": "gAAAAAB-part"},
					{"type": "reasoning", "id": "rs_nested", "summary": [], "encrypted_content": "gAAAAAB-nested"}
				]
			},
			{"type": "function_call_output", "call_id": "call_1", "output": "ok", "encrypted_content": "gAAAAAB-output"}
		]
	}`)

	strippedBody, stripped := stripEncryptedReasoningFromResponsesBody(rawBody)
	require.True(t, stripped)

	inputArr := gjson.GetBytes(strippedBody, "input").Array()
	require.Len(t, inputArr, 3)
	assert.Equal(t, "message", inputArr[0].Get("type").String())
	assert.Equal(t, "message", inputArr[1].Get("type").String())
	assert.Equal(t, "answer", inputArr[1].Get("content.0.text").String())
	assert.False(t, inputArr[1].Get("content.0.encrypted_content").Exists())
	assert.Equal(t, int64(1), inputArr[1].Get("content.#").Int())
	assert.Equal(t, "function_call_output", inputArr[2].Get("type").String())
	assert.Equal(t, "ok", inputArr[2].Get("output").String())
	assert.False(t, inputArr[2].Get("encrypted_content").Exists())
	assert.NotContains(t, string(strippedBody), "encrypted_content")
}

func TestResponsesHelperRetriesAfterStrippingEncryptedReasoning(t *testing.T) {
	gin.SetMode(gin.TestMode)
	prevLogConsume := common.LogConsumeEnabled
	common.LogConsumeEnabled = false
	t.Cleanup(func() {
		common.LogConsumeEnabled = prevLogConsume
	})

	var requestCount int32
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		bodyBytes, err := io.ReadAll(r.Body)
		require.NoError(t, err)

		attempt := atomic.AddInt32(&requestCount, 1)
		if attempt == 1 {
			require.Equal(t, "reasoning", gjson.GetBytes(bodyBytes, "input.1.type").String())
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"error":{"message":"The encrypted content for item rs_0cd7fd73d7f57593016adb1d5791988190b5cbb3e119c6c197 could not be verified. Reason: Encrypted content could not be decrypted or parsed.","type":"invalid_request_error"}}`))
			return
		}

		inputArr := gjson.GetBytes(bodyBytes, "input").Array()
		require.Len(t, inputArr, 2)
		assert.Equal(t, "message", inputArr[0].Get("type").String())
		assert.Equal(t, "function_call_output", inputArr[1].Get("type").String())

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"id": "resp_ok",
			"object": "response",
			"status": "completed",
			"model": "gpt-5.4",
			"output": [
				{
					"type": "message",
					"id": "msg_1",
					"status": "completed",
					"role": "assistant",
					"content": [{"type": "output_text", "text": "done"}]
				}
			],
			"usage": {"input_tokens": 0, "output_tokens": 0, "total_tokens": 0}
		}`))
	}))
	defer upstream.Close()

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", nil)
	c.Request.Header.Set("Content-Type", "application/json")
	common.SetContextKey(c, constant.ContextKeyChannelType, constant.ChannelTypeOpenAI)
	common.SetContextKey(c, constant.ContextKeyChannelBaseUrl, upstream.URL)
	common.SetContextKey(c, constant.ContextKeyOriginalModel, "gpt-5.4")

	req := &dto.OpenAIResponsesRequest{
		Model: "gpt-5.4",
		Input: json.RawMessage(`[
			{"type":"message","role":"user","content":[{"type":"input_text","text":"hi"}]},
			{"id":"rs_0cd7fd73d7f57593016adb1d5791988190b5cbb3e119c6c197","type":"reasoning","encrypted_content":"gAAAAAB..."},
			{"type":"function_call_output","call_id":"call_1","output":"ok"}
		]`),
	}

	info := &relaycommon.RelayInfo{
		RelayMode:       relayconstant.RelayModeResponses,
		RelayFormat:     types.RelayFormatOpenAIResponses,
		OriginModelName: "gpt-5.4",
		RequestURLPath:  "/v1/responses",
		Request:         req,
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelType:       constant.ChannelTypeOpenAI,
			ChannelId:         1,
			ChannelBaseUrl:    upstream.URL,
			UpstreamModelName: "gpt-5.4",
			ApiType:           constant.APITypeOpenAI,
		},
		ResponsesUsageInfo: &relaycommon.ResponsesUsageInfo{
			BuiltInTools: map[string]*relaycommon.BuildInToolInfo{},
		},
	}
	info.PriceData.UsePrice = true

	apiErr := ResponsesHelper(c, info)
	require.Nil(t, apiErr)
	assert.Equal(t, int32(2), atomic.LoadInt32(&requestCount))
	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.Equal(t, "resp_ok", gjson.GetBytes(recorder.Body.Bytes(), "id").String())

	// Also verify info.Request was stripped in-place for any subsequent outer retries.
	var updatedInput []map[string]any
	require.NoError(t, common.Unmarshal(req.Input, &updatedInput))
	require.Len(t, updatedInput, 2)
}

func TestIsUnsupportedSearchContentTypesError(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		err        *types.NewAPIError
		want       bool
	}{
		{
			name:       "web search tools search_content_types not supported",
			statusCode: http.StatusBadRequest,
			err: types.WithOpenAIError(types.OpenAIError{
				Message: "The 'search_content_types' field is not supported for web search tools.",
				Type:    "invalid_request_error",
			}, http.StatusBadRequest),
			want: true,
		},
		{
			name:       "unknown parameter search_content_types",
			statusCode: http.StatusBadRequest,
			err: types.WithOpenAIError(types.OpenAIError{
				Message: "Unknown parameter: 'tools[0].search_content_types'.",
				Type:    "invalid_request_error",
			}, http.StatusBadRequest),
			want: true,
		},
		{
			name:       "extra inputs not permitted for search_content_types",
			statusCode: http.StatusBadRequest,
			err: types.WithOpenAIError(types.OpenAIError{
				Message: "tools.0.search_content_types: Extra inputs are not permitted",
				Type:    "invalid_request_error",
			}, http.StatusBadRequest),
			want: true,
		},
		{
			name:       "unrelated 400 error",
			statusCode: http.StatusBadRequest,
			err: types.WithOpenAIError(types.OpenAIError{
				Message: "Invalid parameter: 'temperature' is not supported with this model.",
				Type:    "invalid_request_error",
			}, http.StatusBadRequest),
			want: false,
		},
		{
			name:       "non-400 status code",
			statusCode: http.StatusInternalServerError,
			err: types.WithOpenAIError(types.OpenAIError{
				Message: "The 'search_content_types' field is not supported for web search tools.",
			}, http.StatusInternalServerError),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, isUnsupportedSearchContentTypesError(tt.statusCode, tt.err))
		})
	}
}

func TestStripSearchContentTypesFromResponsesBody(t *testing.T) {
	rawBody := []byte(`{
		"model": "gpt-6.1-sol",
		"input": [
			{
				"type": "additional_tools",
				"role": "developer",
				"tools": [
					{"type": "web_search", "external_web_access": true, "search_content_types": ["text", "image"]}
				]
			},
			{"type": "message", "role": "user", "content": [{"type": "input_text", "text": "hello"}]}
		],
		"tools": [
			{"type": "web_search", "external_web_access": true, "search_content_types": ["text", "image"]},
			{"type": "function", "name": "exec", "parameters": {}}
		]
	}`)

	strippedBody, stripped := stripSearchContentTypesFromResponsesBody(rawBody)
	require.True(t, stripped)
	assert.Equal(t, "web_search", gjson.GetBytes(strippedBody, "tools.0.type").String())
	assert.True(t, gjson.GetBytes(strippedBody, "tools.0.external_web_access").Bool())
	assert.False(t, gjson.GetBytes(strippedBody, "tools.0.search_content_types").Exists())
	assert.Equal(t, "exec", gjson.GetBytes(strippedBody, "tools.1.name").String())
	assert.False(t, gjson.GetBytes(strippedBody, "input.0.tools.0.search_content_types").Exists())

	_, strippedAgain := stripSearchContentTypesFromResponsesBody(strippedBody)
	assert.False(t, strippedAgain)
}

func TestResponsesHelperRetriesAfterStrippingSearchContentTypes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	prevLogConsume := common.LogConsumeEnabled
	common.LogConsumeEnabled = false
	t.Cleanup(func() {
		common.LogConsumeEnabled = prevLogConsume
	})

	var requestCount int32
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		bodyBytes, err := io.ReadAll(r.Body)
		require.NoError(t, err)

		attempt := atomic.AddInt32(&requestCount, 1)
		if attempt == 1 {
			require.True(t, gjson.GetBytes(bodyBytes, "tools.0.search_content_types").Exists())
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"error":{"message":"The 'search_content_types' field is not supported for web search tools.","type":"invalid_request_error"}}`))
			return
		}

		assert.Equal(t, "web_search", gjson.GetBytes(bodyBytes, "tools.0.type").String())
		assert.True(t, gjson.GetBytes(bodyBytes, "tools.0.external_web_access").Bool())
		assert.False(t, gjson.GetBytes(bodyBytes, "tools.0.search_content_types").Exists())

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"id": "resp_ok",
			"object": "response",
			"status": "completed",
			"model": "gpt-6.1-sol",
			"output": [
				{
					"type": "message",
					"id": "msg_1",
					"status": "completed",
					"role": "assistant",
					"content": [{"type": "output_text", "text": "done"}]
				}
			],
			"usage": {"input_tokens": 0, "output_tokens": 0, "total_tokens": 0}
		}`))
	}))
	defer upstream.Close()

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", nil)
	c.Request.Header.Set("Content-Type", "application/json")
	common.SetContextKey(c, constant.ContextKeyChannelType, constant.ChannelTypeOpenAI)
	common.SetContextKey(c, constant.ContextKeyChannelBaseUrl, upstream.URL)
	common.SetContextKey(c, constant.ContextKeyOriginalModel, "gpt-6.1-sol")

	req := &dto.OpenAIResponsesRequest{
		Model: "gpt-6.1-sol",
		Input: json.RawMessage(`[{"type":"message","role":"user","content":[{"type":"input_text","text":"hi"}]}]`),
		Tools: json.RawMessage(`[{"type":"web_search","external_web_access":true,"search_content_types":["text","image"]}]`),
	}

	info := &relaycommon.RelayInfo{
		RelayMode:       relayconstant.RelayModeResponses,
		RelayFormat:     types.RelayFormatOpenAIResponses,
		OriginModelName: "gpt-6.1-sol",
		RequestURLPath:  "/v1/responses",
		Request:         req,
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelType:       constant.ChannelTypeOpenAI,
			ChannelId:         1,
			ChannelBaseUrl:    upstream.URL,
			UpstreamModelName: "gpt-6.1-sol",
			ApiType:           constant.APITypeOpenAI,
		},
		ResponsesUsageInfo: &relaycommon.ResponsesUsageInfo{
			BuiltInTools: map[string]*relaycommon.BuildInToolInfo{},
		},
	}
	info.PriceData.UsePrice = true

	apiErr := ResponsesHelper(c, info)
	require.Nil(t, apiErr)
	assert.Equal(t, int32(2), atomic.LoadInt32(&requestCount))
	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.Equal(t, "resp_ok", gjson.GetBytes(recorder.Body.Bytes(), "id").String())

	var updatedTools []map[string]any
	require.NoError(t, common.Unmarshal(req.Tools, &updatedTools))
	require.Len(t, updatedTools, 1)
	assert.NotContains(t, updatedTools[0], "search_content_types")
}
