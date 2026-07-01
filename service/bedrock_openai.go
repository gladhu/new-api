package service

import (
	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
)

// ShouldBedrockOpenAIUseResponses reports whether chat/completions should be
// transparently converted to the OpenAI Responses API for Bedrock-hosted OpenAI models.
func ShouldBedrockOpenAIUseResponses(channelType int, model string) bool {
	return channelType == constant.ChannelTypeAws && common.IsBedrockOpenAIModel(model)
}
