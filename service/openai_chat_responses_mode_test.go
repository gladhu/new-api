package service

import (
	"testing"

	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/setting/model_setting"
	"github.com/stretchr/testify/assert"
)

func TestShouldChatCompletionsUseResponsesPolicyMatchesAzureGPT6(t *testing.T) {
	t.Parallel()

	policy := model_setting.ChatCompletionsToResponsesPolicy{
		Enabled:       true,
		AllChannels:   false,
		ChannelTypes:  []int{constant.ChannelTypeAzure},
		ModelPatterns: []string{"^gpt-6"},
	}

	assert.True(t, ShouldChatCompletionsUseResponsesPolicy(policy, 12, constant.ChannelTypeAzure, "gpt-6.1-sol-azure"))
	assert.True(t, ShouldChatCompletionsUseResponsesPolicy(policy, 12, constant.ChannelTypeAzure, "gpt-6-luna-1"))
	assert.False(t, ShouldChatCompletionsUseResponsesPolicy(policy, 12, constant.ChannelTypeAzure, "gpt-4o"))
	assert.False(t, ShouldChatCompletionsUseResponsesPolicy(policy, 12, constant.ChannelTypeOpenAI, "gpt-6.1-sol"))
}
