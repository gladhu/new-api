package aws

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relaykit/dto"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBedrockOpenAIResponsesURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		model string
		want  string
	}{
		{model: "gpt-5.6-terra", want: "https://bedrock-mantle.us-east-1.api.aws/openai/v1/responses"},
		{model: "openai.gpt-5.6-terra", want: "https://bedrock-mantle.us-east-1.api.aws/openai/v1/responses"},
		{model: "global.openai.gpt-5.6-terra", want: "https://bedrock-runtime.us-east-1.amazonaws.com/openai/v1/responses"},
		{model: "us.openai.gpt-5.6-terra", want: "https://bedrock-runtime.us-east-1.amazonaws.com/openai/v1/responses"},
		{model: "in.openai.gpt-5.6-terra", want: "https://bedrock-runtime.us-east-1.amazonaws.com/openai/v1/responses"},
	}
	for _, tt := range tests {
		t.Run(tt.model, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, bedrockOpenAIResponsesURL("us-east-1", tt.model))
		})
	}
}

func TestGetRequestURL_BedrockOpenAIResponses(t *testing.T) {
	t.Parallel()

	adaptor := &Adaptor{}
	info := &relaycommon.RelayInfo{
		RelayMode:       relayconstant.RelayModeResponses,
		OriginModelName: "gpt-5.5",
		ChannelMeta: &relaycommon.ChannelMeta{
			ApiKey:            "bedrock-token|us-east-2",
			UpstreamModelName: "gpt-5.5",
			ChannelOtherSettings: dto.ChannelOtherSettings{
				AwsKeyType: dto.AwsKeyTypeApiKey,
			},
		},
	}

	url, err := adaptor.GetRequestURL(info)
	require.NoError(t, err)
	assert.Equal(t, "https://bedrock-mantle.us-east-2.api.aws/openai/v1/responses", url)
	assert.True(t, adaptor.IsBedrockOpenAI)
}

func TestGetRequestURL_BedrockOpenAIChatCompletionsAlsoUsesResponses(t *testing.T) {
	t.Parallel()

	adaptor := &Adaptor{}
	info := &relaycommon.RelayInfo{
		RelayMode:       relayconstant.RelayModeChatCompletions,
		OriginModelName: "gpt-5.6-luna",
		ChannelMeta: &relaycommon.ChannelMeta{
			ApiKey:            "bedrock-token|us-east-1",
			UpstreamModelName: "openai.gpt-5.6-luna",
			ChannelOtherSettings: dto.ChannelOtherSettings{
				AwsKeyType: dto.AwsKeyTypeApiKey,
			},
		},
	}

	url, err := adaptor.GetRequestURL(info)
	require.NoError(t, err)
	assert.Equal(t, "https://bedrock-mantle.us-east-1.api.aws/openai/v1/responses", url)
	assert.True(t, adaptor.IsBedrockOpenAI)
}

func TestGetRequestURL_BedrockOpenAIRequiresApiKeyMode(t *testing.T) {
	t.Parallel()

	adaptor := &Adaptor{}
	info := &relaycommon.RelayInfo{
		RelayMode:       relayconstant.RelayModeResponses,
		OriginModelName: "openai.gpt-5.4",
		ChannelMeta: &relaycommon.ChannelMeta{
			ApiKey:            "access|secret|us-east-1",
			UpstreamModelName: "openai.gpt-5.4",
			ChannelOtherSettings: dto.ChannelOtherSettings{
				AwsKeyType: dto.AwsKeyTypeAKSK,
			},
		},
	}

	_, err := adaptor.GetRequestURL(info)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "API Key authentication")
}

func TestSetupRequestHeader_BedrockOpenAIUsesBearerTokenOnly(t *testing.T) {
	t.Parallel()

	adaptor := &Adaptor{IsBedrockOpenAI: true}
	info := &relaycommon.RelayInfo{
		RelayMode: relayconstant.RelayModeResponses,
		ChannelMeta: &relaycommon.ChannelMeta{
			ApiKey: "bedrock-token|us-west-2",
		},
	}
	headers := http.Header{}
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", nil)
	c.Request.Header.Set("Content-Type", "application/json")

	err := adaptor.SetupRequestHeader(c, &headers, info)
	require.NoError(t, err)
	assert.Equal(t, "Bearer bedrock-token", headers.Get("Authorization"))
}

func TestConvertOpenAIResponsesRequest_BedrockOpenAIModelMapping(t *testing.T) {
	t.Parallel()

	adaptor := &Adaptor{}
	info := &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: "gpt-5.4",
		},
	}

	converted, err := adaptor.ConvertOpenAIResponsesRequest(nil, info, dto.OpenAIResponsesRequest{
		Model: "gpt-5.4",
		Input: json.RawMessage(`"hello"`),
	})
	require.NoError(t, err)

	responsesReq, ok := converted.(dto.OpenAIResponsesRequest)
	require.True(t, ok)
	assert.Equal(t, "openai.gpt-5.4", responsesReq.Model)
	assert.Equal(t, "openai.gpt-5.4", info.UpstreamModelName)
	assert.True(t, adaptor.IsBedrockOpenAI)
}

func TestConvertOpenAIResponsesRequest_BedrockOpenAIReasoningEffortSuffix(t *testing.T) {
	t.Parallel()

	adaptor := &Adaptor{}
	info := &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: "gpt-5.5",
		},
	}

	converted, err := adaptor.ConvertOpenAIResponsesRequest(nil, info, dto.OpenAIResponsesRequest{
		Model: "gpt-5.5-high",
		Input: json.RawMessage(`"hello"`),
	})
	require.NoError(t, err)

	responsesReq, ok := converted.(dto.OpenAIResponsesRequest)
	require.True(t, ok)
	assert.Equal(t, "openai.gpt-5.5", responsesReq.Model)
	require.NotNil(t, responsesReq.Reasoning)
	assert.Equal(t, "high", responsesReq.Reasoning.Effort)
	assert.Equal(t, "high", info.ReasoningEffort)
}

func TestConvertOpenAIResponsesRequest_BedrockMapsGpt56Family(t *testing.T) {
	t.Parallel()

	tests := []struct {
		in   string
		want string
	}{
		{in: "gpt-5.6-luna", want: "openai.gpt-5.6-luna"},
		{in: "gpt-5.6-sol", want: "openai.gpt-5.6-sol"},
		{in: "gpt-5.6-terra", want: "openai.gpt-5.6-terra"},
		{in: "openai.gpt-5.6-luna", want: "openai.gpt-5.6-luna"},
		{in: "global.openai.gpt-5.6-terra", want: "global.openai.gpt-5.6-terra"},
		{in: "us.openai.gpt-5.6-terra", want: "us.openai.gpt-5.6-terra"},
		{in: "global.gpt-5.6-terra", want: "global.openai.gpt-5.6-terra"},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			t.Parallel()
			adaptor := &Adaptor{}
			info := &relaycommon.RelayInfo{
				ChannelMeta: &relaycommon.ChannelMeta{UpstreamModelName: tt.in},
			}
			converted, err := adaptor.ConvertOpenAIResponsesRequest(nil, info, dto.OpenAIResponsesRequest{
				Model: tt.in,
				Input: json.RawMessage(`"hello"`),
			})
			require.NoError(t, err)
			responsesReq, ok := converted.(dto.OpenAIResponsesRequest)
			require.True(t, ok)
			assert.Equal(t, tt.want, responsesReq.Model)
			assert.Equal(t, tt.want, info.UpstreamModelName)
		})
	}
}

func TestNormalizeCodexResponsesLiteForBedrock(t *testing.T) {
	t.Parallel()

	req := dto.OpenAIResponsesRequest{
		Model: "gpt-5.6-luna",
		Input: json.RawMessage(`[
			{"type":"additional_tools","role":"developer","tools":[
				{"type":"custom","name":"exec","description":"run js"},
				{"type":"function","name":"collaboration.spawn","description":"skip me"},
				{"type":"function","name":"wait","description":"wait"}
			]},
			{"type":"message","role":"user","content":[{"type":"input_text","text":"hi"}]}
		]`),
		Reasoning: &dto.Reasoning{
			Effort:  "medium",
			Context: json.RawMessage(`"all_turns"`),
		},
	}

	require.NoError(t, normalizeCodexResponsesLiteForBedrock(&req))
	require.NotNil(t, req.Reasoning)
	assert.Equal(t, "medium", req.Reasoning.Effort)
	assert.Empty(t, req.Reasoning.Context)

	var input []map[string]any
	require.NoError(t, json.Unmarshal(req.Input, &input))
	require.Len(t, input, 1)
	assert.Equal(t, "message", input[0]["type"])

	var tools []map[string]any
	require.NoError(t, json.Unmarshal(req.Tools, &tools))
	require.Len(t, tools, 2)
	assert.Equal(t, "exec", tools[0]["name"])
	assert.Equal(t, "wait", tools[1]["name"])
}

func TestConvertOpenAIResponsesRequest_BedrockNormalizesLiteEndToEnd(t *testing.T) {
	t.Parallel()

	adaptor := &Adaptor{}
	info := &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{UpstreamModelName: "gpt-5.6-luna"},
	}
	converted, err := adaptor.ConvertOpenAIResponsesRequest(nil, info, dto.OpenAIResponsesRequest{
		Model: "gpt-5.6-luna",
		Input: json.RawMessage(`[
			{"type":"additional_tools","role":"developer","tools":[{"type":"function","name":"wait"}]},
			{"type":"message","role":"user","content":[{"type":"input_text","text":"hi"}]}
		]`),
		Reasoning: &dto.Reasoning{Effort: "low", Context: json.RawMessage(`"all_turns"`)},
	})
	require.NoError(t, err)
	responsesReq, ok := converted.(dto.OpenAIResponsesRequest)
	require.True(t, ok)
	assert.Equal(t, "openai.gpt-5.6-luna", responsesReq.Model)
	assert.Empty(t, responsesReq.Reasoning.Context)

	var input []map[string]any
	require.NoError(t, json.Unmarshal(responsesReq.Input, &input))
	require.Len(t, input, 1)
	assert.Equal(t, "message", input[0]["type"])

	var tools []map[string]any
	require.NoError(t, json.Unmarshal(responsesReq.Tools, &tools))
	require.Len(t, tools, 1)
	assert.Equal(t, "wait", tools[0]["name"])
}

func TestGetRequestURL_ClaudeApiKeyModeUsesCorrectRegionOrder(t *testing.T) {
	t.Parallel()

	adaptor := &Adaptor{}
	info := &relaycommon.RelayInfo{
		RelayMode: relayconstant.RelayModeChatCompletions,
		ChannelMeta: &relaycommon.ChannelMeta{
			ApiKey:            "bedrock-token|ap-northeast-1",
			UpstreamModelName: "claude-sonnet-4-6",
			ChannelOtherSettings: dto.ChannelOtherSettings{
				AwsKeyType: dto.AwsKeyTypeApiKey,
			},
		},
	}

	url, err := adaptor.GetRequestURL(info)
	require.NoError(t, err)
	assert.Equal(t, "https://bedrock-runtime.ap-northeast-1.amazonaws.com/model/anthropic.claude-sonnet-4-6/converse", url)
}

func TestConvertOpenAIRequest_BedrockOpenAIChatCompletionsToResponses(t *testing.T) {
	t.Parallel()

	adaptor := &Adaptor{}
	info := &relaycommon.RelayInfo{
		RelayMode:       relayconstant.RelayModeChatCompletions,
		OriginModelName: "gpt-5.6-luna",
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: "gpt-5.6-luna",
		},
	}
	stream := false
	req := &dto.GeneralOpenAIRequest{
		Model:  "gpt-5.6-luna",
		Stream: &stream,
		Messages: []dto.Message{
			{Role: "user", Content: "hello"},
		},
	}

	converted, err := adaptor.ConvertOpenAIRequest(nil, info, req)
	require.NoError(t, err)
	responsesReq, ok := converted.(dto.OpenAIResponsesRequest)
	require.True(t, ok)
	assert.Equal(t, "openai.gpt-5.6-luna", responsesReq.Model)
	assert.Equal(t, "openai.gpt-5.6-luna", info.UpstreamModelName)
	assert.Equal(t, relayconstant.RelayModeResponses, info.RelayMode)
	assert.True(t, adaptor.IsBedrockOpenAI)
	assert.NotEmpty(t, responsesReq.Input)
}

func TestConvertOpenAIResponsesRequest_PreservesMappedGlobalInferenceProfile(t *testing.T) {
	t.Parallel()

	adaptor := &Adaptor{}
	info := &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: "global.openai.gpt-5.6-terra",
		},
	}

	converted, err := adaptor.ConvertOpenAIResponsesRequest(nil, info, dto.OpenAIResponsesRequest{
		Model: "global.openai.gpt-5.6-terra",
		Input: json.RawMessage(`"hello"`),
	})
	require.NoError(t, err)

	responsesReq, ok := converted.(dto.OpenAIResponsesRequest)
	require.True(t, ok)
	assert.Equal(t, "global.openai.gpt-5.6-terra", responsesReq.Model)
	assert.Equal(t, "global.openai.gpt-5.6-terra", info.UpstreamModelName)
	assert.True(t, adaptor.IsBedrockOpenAI)
}

func TestConvertOpenAIResponsesRequest_PrefersMappedGlobalProfileOverFriendlyName(t *testing.T) {
	t.Parallel()

	adaptor := &Adaptor{}
	info := &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: "global.openai.gpt-5.6-terra",
		},
	}

	converted, err := adaptor.ConvertOpenAIResponsesRequest(nil, info, dto.OpenAIResponsesRequest{
		Model: "gpt-5.6-terra",
		Input: json.RawMessage(`"hello"`),
	})
	require.NoError(t, err)

	responsesReq, ok := converted.(dto.OpenAIResponsesRequest)
	require.True(t, ok)
	assert.Equal(t, "global.openai.gpt-5.6-terra", responsesReq.Model)
	assert.Equal(t, "global.openai.gpt-5.6-terra", info.UpstreamModelName)
}

func TestGetRequestURL_BedrockOpenAIGlobalMappedModel(t *testing.T) {
	t.Parallel()

	adaptor := &Adaptor{}
	info := &relaycommon.RelayInfo{
		RelayMode:       relayconstant.RelayModeResponses,
		OriginModelName: "gpt-5.6-terra",
		ChannelMeta: &relaycommon.ChannelMeta{
			ApiKey:            "bedrock-token|us-east-1",
			UpstreamModelName: "global.openai.gpt-5.6-terra",
			ChannelOtherSettings: dto.ChannelOtherSettings{
				AwsKeyType: dto.AwsKeyTypeApiKey,
			},
		},
	}

	url, err := adaptor.GetRequestURL(info)
	require.NoError(t, err)
	assert.Equal(t, "https://bedrock-runtime.us-east-1.amazonaws.com/openai/v1/responses", url)
	assert.True(t, adaptor.IsBedrockOpenAI)
}

func TestGetRequestURL_BedrockOpenAIUSMappedModel(t *testing.T) {
	t.Parallel()

	adaptor := &Adaptor{}
	info := &relaycommon.RelayInfo{
		RelayMode:       relayconstant.RelayModeResponses,
		OriginModelName: "gpt-5.6-terra",
		ChannelMeta: &relaycommon.ChannelMeta{
			ApiKey:            "bedrock-token|us-west-2",
			UpstreamModelName: "us.openai.gpt-5.6-terra",
			ChannelOtherSettings: dto.ChannelOtherSettings{
				AwsKeyType: dto.AwsKeyTypeApiKey,
			},
		},
	}

	url, err := adaptor.GetRequestURL(info)
	require.NoError(t, err)
	assert.Equal(t, "https://bedrock-runtime.us-west-2.amazonaws.com/openai/v1/responses", url)
	assert.True(t, adaptor.IsBedrockOpenAI)
}

func TestGetModelList_IncludesBedrockOpenAIModels(t *testing.T) {
	t.Parallel()

	models := (&Adaptor{}).GetModelList()
	require.Contains(t, models, "gpt-5.4")
	require.Contains(t, models, "gpt-5.5")
	require.Contains(t, models, "gpt-5.6")
	require.Contains(t, models, "gpt-5.6-luna")
	require.Contains(t, models, "gpt-5.6-sol")
	require.Contains(t, models, "gpt-5.6-terra")
}
