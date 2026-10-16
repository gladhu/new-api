package service

import (
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
)

// ShouldAzureOpenAIChatCompletionsCompat reports whether an incoming
// /v1/chat/completions request must be converted to the OpenAI Responses API.
//
// Azure GPT-6+ deployments reject function tools on /v1/chat/completions when
// reasoning is enabled (the default). The same requests succeed on
// /openai/v1/responses, matching the Bedrock Mantle OpenAI compatibility path.
func ShouldAzureOpenAIChatCompletionsCompat(channelType int, models ...string) bool {
	if channelType != constant.ChannelTypeAzure {
		return false
	}
	for _, model := range models {
		if common.IsOpenAIGPT6PlusModel(model) {
			return true
		}
	}
	return false
}

var azureGPT6SupportedReasoningEfforts = map[string]struct{}{
	"low":    {},
	"medium": {},
	"high":   {},
	"xhigh":  {},
	"max":    {},
}

// NormalizeAzureGPT6ReasoningEffort maps client effort onto values Azure GPT-6
// accepts on the Responses API. Empty stays empty so the field can be omitted
// and the upstream default used. Unsupported values including none and minimal
// collapse to high for better quality.
func NormalizeAzureGPT6ReasoningEffort(effort string) string {
	effort = strings.ToLower(strings.TrimSpace(effort))
	if effort == "" {
		return ""
	}
	if _, ok := azureGPT6SupportedReasoningEfforts[effort]; ok {
		return effort
	}
	return "high"
}
