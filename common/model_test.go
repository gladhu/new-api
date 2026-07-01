package common

import "testing"

func TestIsBedrockOpenAIModel(t *testing.T) {
	t.Parallel()

	tests := []struct {
		model string
		want  bool
	}{
		{model: "gpt-5.4", want: true},
		{model: "gpt-5.5", want: true},
		{model: "gpt-5.6", want: true},
		{model: "gpt-5.5-high", want: true},
		{model: "openai.gpt-5.4", want: true},
		{model: "openai.gpt-5.5", want: true},
		{model: "openai.gpt-5.6", want: true},
		{model: "claude-sonnet-4-6", want: false},
		{model: "gpt-4o", want: false},
		{model: "openai.gpt-oss-120b-1:0", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.model, func(t *testing.T) {
			t.Parallel()
			if got := IsBedrockOpenAIModel(tt.model); got != tt.want {
				t.Fatalf("IsBedrockOpenAIModel(%q) = %v, want %v", tt.model, got, tt.want)
			}
		})
	}
}
