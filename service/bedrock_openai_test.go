package service

import (
	"testing"

	"github.com/QuantumNous/new-api/constant"
	"github.com/stretchr/testify/assert"
)

func TestShouldBedrockOpenAIChatCompletionsCompat(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		channelType    int
		apiType        int
		channelBaseURL string
		models         []string
		want           bool
	}{
		{
			name:        "aws channel with luna",
			channelType: constant.ChannelTypeAws,
			apiType:     constant.APITypeAws,
			models:      []string{"gpt-5.6-luna"},
			want:        true,
		},
		{
			name:        "aws channel with openai-prefixed model",
			channelType: constant.ChannelTypeAws,
			apiType:     constant.APITypeAws,
			models:      []string{"openai.gpt-5.6-luna"},
			want:        true,
		},
		{
			name:        "aws channel with global inference profile",
			channelType: constant.ChannelTypeAws,
			apiType:     constant.APITypeAws,
			models:      []string{"global.openai.gpt-5.6-terra"},
			want:        true,
		},
		{
			name:        "aws channel with us inference profile",
			channelType: constant.ChannelTypeAws,
			apiType:     constant.APITypeAws,
			models:      []string{"us.openai.gpt-5.6-terra"},
			want:        true,
		},
		{
			name:        "aws channel with custom alias and upstream model",
			channelType: constant.ChannelTypeAws,
			apiType:     constant.APITypeAws,
			models:      []string{"my-luna", "openai.gpt-5.6-luna"},
			want:        true,
		},
		{
			name:        "aws api type alone with luna",
			channelType: 0,
			apiType:     constant.APITypeAws,
			models:      []string{"gpt-5.6-luna"},
			want:        true,
		},
		{
			name:        "aws channel with claude model",
			channelType: constant.ChannelTypeAws,
			apiType:     constant.APITypeAws,
			models:      []string{"claude-sonnet-4-6"},
			want:        false,
		},
		{
			name:           "openai channel pointed at mantle",
			channelType:    constant.ChannelTypeOpenAI,
			apiType:        constant.APITypeOpenAI,
			channelBaseURL: "https://bedrock-mantle.us-east-1.api.aws/openai",
			models:         []string{"openai.gpt-5.6-luna"},
			want:           true,
		},
		{
			name:           "openai channel pointed at real openai",
			channelType:    constant.ChannelTypeOpenAI,
			apiType:        constant.APITypeOpenAI,
			channelBaseURL: "https://api.openai.com",
			models:         []string{"gpt-5.6"},
			want:           false,
		},
		{
			name:        "non-aws without mantle base url",
			channelType: constant.ChannelTypeAnthropic,
			apiType:     constant.APITypeAnthropic,
			models:      []string{"gpt-5.6-luna"},
			want:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := ShouldBedrockOpenAIChatCompletionsCompat(tt.channelType, tt.apiType, tt.channelBaseURL, tt.models...)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestShouldBedrockOpenAIUseResponses(t *testing.T) {
	t.Parallel()

	assert.True(t, ShouldBedrockOpenAIUseResponses(constant.ChannelTypeAws, "gpt-5.6-luna"))
	assert.False(t, ShouldBedrockOpenAIUseResponses(constant.ChannelTypeOpenAI, "gpt-5.6-luna"))
	assert.False(t, ShouldBedrockOpenAIUseResponses(constant.ChannelTypeAws, "claude-sonnet-4-6"))
}
