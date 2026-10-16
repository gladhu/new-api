package service

import (
	"testing"

	"github.com/QuantumNous/new-api/constant"
	"github.com/stretchr/testify/assert"
)

func TestShouldAzureOpenAIChatCompletionsCompat(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		channelType int
		models      []string
		want        bool
	}{
		{
			name:        "azure gpt-6.1-sol",
			channelType: constant.ChannelTypeAzure,
			models:      []string{"gpt-6.1-sol"},
			want:        true,
		},
		{
			name:        "azure gpt-6.1-sol-azure alias",
			channelType: constant.ChannelTypeAzure,
			models:      []string{"gpt-6.1-sol-azure"},
			want:        true,
		},
		{
			name:        "azure gpt-6-luna-1",
			channelType: constant.ChannelTypeAzure,
			models:      []string{"gpt-6-luna-1"},
			want:        true,
		},
		{
			name:        "azure alias mapped to gpt-6 upstream",
			channelType: constant.ChannelTypeAzure,
			models:      []string{"my-luna", "gpt-6-luna-1"},
			want:        true,
		},
		{
			name:        "azure gpt-4o stays on chat completions",
			channelType: constant.ChannelTypeAzure,
			models:      []string{"gpt-4o"},
			want:        false,
		},
		{
			name:        "azure gpt-5 stays on chat completions",
			channelType: constant.ChannelTypeAzure,
			models:      []string{"gpt-5.1"},
			want:        false,
		},
		{
			name:        "openai channel with gpt-6 is not auto-converted",
			channelType: constant.ChannelTypeOpenAI,
			models:      []string{"gpt-6.1-sol"},
			want:        false,
		},
		{
			name:        "aws channel is handled by bedrock compat",
			channelType: constant.ChannelTypeAws,
			models:      []string{"gpt-6-astra"},
			want:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := ShouldAzureOpenAIChatCompletionsCompat(tt.channelType, tt.models...)
			assert.Equal(t, tt.want, got)
		})
	}
}
