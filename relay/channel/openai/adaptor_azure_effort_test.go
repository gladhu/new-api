package openai

import (
	"testing"

	"github.com/QuantumNous/new-api/constant"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConvertOpenAIResponsesRequestNormalizesAzureGPT6Effort(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		channelType  int
		model        string
		effort       string
		wantEffort   string
		wantRecorded string
	}{
		{
			name:         "azure gpt-6 none becomes high",
			channelType:  constant.ChannelTypeAzure,
			model:        "gpt-6.1-sol",
			effort:       "none",
			wantEffort:   "high",
			wantRecorded: "high",
		},
		{
			name:         "azure gpt-6 minimal becomes high",
			channelType:  constant.ChannelTypeAzure,
			model:        "gpt-6.1-sol-azure",
			effort:       "minimal",
			wantEffort:   "high",
			wantRecorded: "high",
		},
		{
			name:         "azure gpt-6 xhigh is kept",
			channelType:  constant.ChannelTypeAzure,
			model:        "gpt-6-luna-1",
			effort:       "xhigh",
			wantEffort:   "xhigh",
			wantRecorded: "xhigh",
		},
		{
			name:         "azure gpt-6 suffix none becomes high",
			channelType:  constant.ChannelTypeAzure,
			model:        "gpt-6.1-sol-none",
			effort:       "",
			wantEffort:   "high",
			wantRecorded: "high",
		},
		{
			name:         "openai channel keeps none",
			channelType:  constant.ChannelTypeOpenAI,
			model:        "gpt-6.1-sol",
			effort:       "none",
			wantEffort:   "none",
			wantRecorded: "none",
		},
		{
			name:        "azure gpt-4o keeps none",
			channelType: constant.ChannelTypeAzure,
			model:       "gpt-4o",
			effort:      "none",
			wantEffort:  "none",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			info := &relaycommon.RelayInfo{
				ChannelMeta: &relaycommon.ChannelMeta{
					ChannelType:       tt.channelType,
					UpstreamModelName: tt.model,
				},
				OriginModelName: tt.model,
			}
			req := dto.OpenAIResponsesRequest{Model: tt.model}
			if tt.effort != "" {
				req.Reasoning = &dto.Reasoning{Effort: tt.effort}
			}

			converted, err := (&Adaptor{}).ConvertOpenAIResponsesRequest(nil, info, req)
			require.NoError(t, err)
			got, ok := converted.(dto.OpenAIResponsesRequest)
			require.True(t, ok)
			require.NotNil(t, got.Reasoning)
			assert.Equal(t, tt.wantEffort, got.Reasoning.Effort)
			if tt.wantRecorded != "" {
				assert.Equal(t, tt.wantRecorded, info.ReasoningEffort)
			}
		})
	}
}

func TestConvertOpenAIResponsesRequestLeavesAzureGPT6OmittedEffort(t *testing.T) {
	t.Parallel()

	info := &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelType:       constant.ChannelTypeAzure,
			UpstreamModelName: "gpt-6.1-sol",
		},
		OriginModelName: "gpt-6.1-sol",
	}
	converted, err := (&Adaptor{}).ConvertOpenAIResponsesRequest(nil, info, dto.OpenAIResponsesRequest{
		Model: "gpt-6.1-sol",
	})
	require.NoError(t, err)
	got, ok := converted.(dto.OpenAIResponsesRequest)
	require.True(t, ok)
	assert.Nil(t, got.Reasoning)
	assert.Empty(t, info.ReasoningEffort)
}
