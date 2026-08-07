package service

import (
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
)

// ShouldBedrockOpenAIUseResponses reports whether chat/completions should be
// transparently converted to the OpenAI Responses API for Bedrock-hosted OpenAI models.
func ShouldBedrockOpenAIUseResponses(channelType int, model string) bool {
	return ShouldBedrockOpenAIChatCompletionsCompat(channelType, -1, "", model)
}

// ShouldBedrockOpenAIChatCompletionsCompat reports whether an incoming
// /v1/chat/completions request must be converted to the OpenAI Responses API.
//
// Bedrock Mantle OpenAI models (GPT-5.4/5.5/5.6) only support /openai/v1/responses.
// This covers:
//   - AWS channels (type 33 / APITypeAws), which always route Mantle OpenAI models to responses
//   - OpenAI-compatible channels whose base URL points at bedrock-mantle
func ShouldBedrockOpenAIChatCompletionsCompat(channelType int, apiType int, channelBaseURL string, models ...string) bool {
	matched := false
	for _, model := range models {
		if common.IsBedrockOpenAIModel(model) {
			matched = true
			break
		}
	}
	if !matched {
		return false
	}
	if channelType == constant.ChannelTypeAws || apiType == constant.APITypeAws {
		return true
	}
	return strings.Contains(strings.ToLower(channelBaseURL), "bedrock-mantle")
}
