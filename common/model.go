package common

import "strings"

var (
	// OpenAIResponseOnlyModels is a list of models that are only available for OpenAI responses.
	OpenAIResponseOnlyModels = []string{
		"o3-pro",
		"o3-deep-research",
		"o4-mini-deep-research",
	}
	ImageGenerationModels = []string{
		"dall-e-3",
		"dall-e-2",
		"gpt-image-1",
		"prefix:imagen-",
		"flux-",
		"flux.1-",
	}
	OpenAITextModels = []string{
		"gpt-",
		"o1",
		"o3",
		"o4",
		"chatgpt",
	}
)

func IsOpenAIResponseOnlyModel(modelName string) bool {
	for _, m := range OpenAIResponseOnlyModels {
		if strings.Contains(modelName, m) {
			return true
		}
	}
	return false
}

func IsImageGenerationModel(modelName string) bool {
	modelName = strings.ToLower(modelName)
	for _, m := range ImageGenerationModels {
		if strings.Contains(modelName, m) {
			return true
		}
		if strings.HasPrefix(m, "prefix:") && strings.HasPrefix(modelName, strings.TrimPrefix(m, "prefix:")) {
			return true
		}
	}
	return false
}

func IsOpenAITextModel(modelName string) bool {
	modelName = strings.ToLower(modelName)
	for _, m := range OpenAITextModels {
		if strings.Contains(modelName, m) {
			return true
		}
	}
	return false
}

// IsBedrockOpenAIModel reports whether the model is an OpenAI frontier model hosted on AWS Bedrock
// (GPT-5.4 / GPT-5.5 / GPT-5.6), which only supports the OpenAI Responses API via bedrock-mantle endpoints.
func IsBedrockOpenAIModel(modelName string) bool {
	modelName = strings.ToLower(strings.TrimSpace(modelName))
	switch {
	case strings.HasPrefix(modelName, "openai.gpt-5.4"):
		return true
	case strings.HasPrefix(modelName, "openai.gpt-5.5"):
		return true
	case strings.HasPrefix(modelName, "openai.gpt-5.6"):
		return true
	case strings.HasPrefix(modelName, "gpt-5.4"):
		return true
	case strings.HasPrefix(modelName, "gpt-5.5"):
		return true
	case strings.HasPrefix(modelName, "gpt-5.6"):
		return true
	default:
		return false
	}
}
