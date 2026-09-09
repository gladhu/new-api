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

// IsBedrockOpenAIModel reports whether the model is an OpenAI frontier model
// hosted on AWS Bedrock (Responses API via bedrock-mantle / bedrock-runtime).
//
// Matching is version-based so new Bedrock OpenAI models do not need a
// per-ID whitelist:
//   - gpt-5.4 and later 5.x (gpt-5.4, gpt-5.5, gpt-5.6-luna, ...)
//   - gpt-6 and later generations (gpt-6-astra, ...)
// CRIS prefixes (global./us./...) and the openai. provider prefix are ignored.
// gpt-oss remains excluded (different Bedrock product, not Mantle Responses).
func IsBedrockOpenAIModel(modelName string) bool {
	_, modelName = SplitBedrockInferenceProfile(modelName)
	modelName = strings.TrimPrefix(modelName, "openai.")
	if !strings.HasPrefix(modelName, "gpt-") || strings.HasPrefix(modelName, "gpt-oss") {
		return false
	}

	rest := modelName[len("gpt-"):]
	major, i := 0, 0
	for i < len(rest) && rest[i] >= '0' && rest[i] <= '9' {
		major = major*10 + int(rest[i]-'0')
		i++
	}
	if i == 0 {
		return false
	}
	if major >= 6 {
		return true
	}
	if major != 5 || i >= len(rest) || rest[i] != '.' {
		return false
	}
	minor, j := 0, i+1
	for j < len(rest) && rest[j] >= '0' && rest[j] <= '9' {
		minor = minor*10 + int(rest[j]-'0')
		j++
	}
	return j > i+1 && minor >= 4
}
