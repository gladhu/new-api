package aws

import (
	"fmt"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/QuantumNous/new-api/setting/reasoning"
	"github.com/pkg/errors"
)

func parseAwsApiKeyAndRegion(apiKey string) (token, region string, err error) {
	parts := strings.Split(apiKey, "|")
	if len(parts) != 2 {
		return "", "", errors.New("invalid aws api key, should be in format of <api-key>|<region>")
	}
	token = strings.TrimSpace(parts[0])
	region = strings.TrimSpace(parts[1])
	if token == "" || region == "" {
		return "", "", errors.New("invalid aws api key, should be in format of <api-key>|<region>")
	}
	return token, region, nil
}

func bedrockMantleResponsesURL(region string) string {
	return fmt.Sprintf("https://bedrock-mantle.%s.api.aws/openai/v1/responses", region)
}

func convertBedrockOpenAIResponsesRequest(infoModel string, request dto.OpenAIResponsesRequest) (dto.OpenAIResponsesRequest, string, error) {
	if !common.IsBedrockOpenAIModel(request.Model) && !common.IsBedrockOpenAIModel(infoModel) {
		return request, "", errors.New("not a Bedrock OpenAI model")
	}

	// Codex CLI forces Responses Lite for gpt-5.6-*; Bedrock Mantle expects classic Responses.
	if err := normalizeCodexResponsesLiteForBedrock(&request); err != nil {
		return request, "", err
	}

	effort, originModel := reasoning.ParseOpenAIReasoningEffortFromModelSuffix(request.Model)
	if effort != "" {
		if request.Reasoning == nil {
			request.Reasoning = &dto.Reasoning{Effort: effort}
		} else {
			request.Reasoning.Effort = effort
		}
		request.Model = originModel
	}

	// Channel model mapping may already point at a CRIS ID such as
	// global.openai.gpt-5.6-terra. Prefer that over rewriting the friendly name
	// back to openai.gpt-5.6-terra (US / in-region, ~1.1x).
	if prefix, _ := common.SplitBedrockInferenceProfile(infoModel); prefix != "" {
		request.Model = infoModel
	}

	request.Model = getAwsModelID(request.Model)
	return request, effort, nil
}

// normalizeCodexResponsesLiteForBedrock rewrites Codex Responses-Lite bodies into the
// classic Responses shape accepted by Bedrock Mantle OpenAI endpoints.
//
// Lite places tools under input[].type=additional_tools and may set reasoning.context.
// Bedrock rejects that input variant with validation_error.
func normalizeCodexResponsesLiteForBedrock(request *dto.OpenAIResponsesRequest) error {
	if request == nil {
		return nil
	}

	if request.Reasoning != nil && len(request.Reasoning.Context) > 0 {
		request.Reasoning.Context = nil
	}

	if len(request.Input) == 0 || common.GetJsonType(request.Input) != "array" {
		return nil
	}

	var items []any
	if err := common.Unmarshal(request.Input, &items); err != nil {
		return errors.Wrap(err, "parse responses input")
	}

	kept := make([]any, 0, len(items))
	extractedTools := make([]any, 0)
	changed := false
	for _, item := range items {
		m, ok := item.(map[string]any)
		if !ok {
			kept = append(kept, item)
			continue
		}
		typ, _ := m["type"].(string)
		if typ != "additional_tools" {
			kept = append(kept, item)
			continue
		}
		changed = true
		if tools, ok := m["tools"].([]any); ok {
			for _, tool := range tools {
				if isBedrockUnsupportedCodexTool(tool) {
					continue
				}
				extractedTools = append(extractedTools, tool)
			}
		}
	}
	if !changed {
		return nil
	}

	inputBytes, err := common.Marshal(kept)
	if err != nil {
		return errors.Wrap(err, "marshal normalized responses input")
	}
	request.Input = inputBytes

	if len(extractedTools) == 0 {
		return nil
	}

	mergedTools := make([]any, 0, len(extractedTools))
	if len(request.Tools) > 0 && common.GetJsonType(request.Tools) == "array" {
		var existing []any
		if err := common.Unmarshal(request.Tools, &existing); err != nil {
			return errors.Wrap(err, "parse existing responses tools")
		}
		for _, tool := range existing {
			if isBedrockUnsupportedCodexTool(tool) {
				continue
			}
			mergedTools = append(mergedTools, tool)
		}
	}
	mergedTools = append(mergedTools, extractedTools...)
	toolsBytes, err := common.Marshal(mergedTools)
	if err != nil {
		return errors.Wrap(err, "marshal normalized responses tools")
	}
	request.Tools = toolsBytes
	return nil
}

func isBedrockUnsupportedCodexTool(tool any) bool {
	m, ok := tool.(map[string]any)
	if !ok {
		return false
	}
	name, _ := m["name"].(string)
	name = strings.ToLower(strings.TrimSpace(name))
	if strings.HasPrefix(name, "collaboration") || strings.Contains(name, "collaboration.") {
		return true
	}
	typ, _ := m["type"].(string)
	return strings.EqualFold(strings.TrimSpace(typ), "tool_search")
}
