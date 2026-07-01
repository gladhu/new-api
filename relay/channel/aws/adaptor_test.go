package aws

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetRequestURL_BedrockOpenAIResponses(t *testing.T) {
	t.Parallel()

	adaptor := &Adaptor{}
	info := &relaycommon.RelayInfo{
		RelayMode: relayconstant.RelayModeResponses,
		ChannelMeta: &relaycommon.ChannelMeta{
			ApiKey:            "bedrock-token|us-east-2",
			UpstreamModelName: "gpt-5.5",
			OriginModelName:   "gpt-5.5",
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

func TestGetRequestURL_BedrockOpenAIRequiresApiKeyMode(t *testing.T) {
	t.Parallel()

	adaptor := &Adaptor{}
	info := &relaycommon.RelayInfo{
		RelayMode: relayconstant.RelayModeResponses,
		ChannelMeta: &relaycommon.ChannelMeta{
			ApiKey:            "access|secret|us-east-1",
			UpstreamModelName: "openai.gpt-5.4",
			OriginModelName:   "openai.gpt-5.4",
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

	err := adaptor.SetupRequestHeader(nil, &headers, info)
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

func TestGetModelList_IncludesBedrockOpenAIModels(t *testing.T) {
	t.Parallel()

	models := (&Adaptor{}).GetModelList()
	require.Contains(t, models, "gpt-5.4")
	require.Contains(t, models, "gpt-5.5")
	require.Contains(t, models, "gpt-5.6")
}
