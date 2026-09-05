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

// bedrockInferenceProfilePrefixes are AWS cross-Region inference profile
// prefixes. Global CRIS (~list price) uses "global."; geographic CRIS (US is
// about 1.1x) uses a geography code such as "us.".
var bedrockInferenceProfilePrefixes = []string{
	"global.",
	"us.",
	"eu.",
	"apac.",
	"jp.",
	"au.",
	"in.",
}

// SplitBedrockInferenceProfile separates an AWS CRIS prefix from a model ID.
// Example: "global.openai.gpt-5.6-terra" → ("global.", "openai.gpt-5.6-terra").
func SplitBedrockInferenceProfile(modelName string) (prefix, base string) {
	modelName = strings.ToLower(strings.TrimSpace(modelName))
	for _, p := range bedrockInferenceProfilePrefixes {
		if strings.HasPrefix(modelName, p) {
			return p, strings.TrimPrefix(modelName, p)
		}
	}
	return "", modelName
}

// IsBedrockOpenAIModel reports whether the model is an OpenAI frontier model hosted on AWS Bedrock
// (GPT-5.4 / GPT-5.5 / GPT-5.6). Unprefixed IDs use bedrock-mantle; CRIS IDs such as
// global.openai.gpt-5.6-terra use bedrock-runtime.
func IsBedrockOpenAIModel(modelName string) bool {
	_, modelName = SplitBedrockInferenceProfile(modelName)
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
