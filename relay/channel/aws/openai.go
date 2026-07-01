package aws

import (
	"fmt"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
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

	effort, originModel := reasoning.ParseOpenAIReasoningEffortFromModelSuffix(request.Model)
	if effort != "" {
		if request.Reasoning == nil {
			request.Reasoning = &dto.Reasoning{Effort: effort}
		} else {
			request.Reasoning.Effort = effort
		}
		request.Model = originModel
	}

	request.Model = getAwsModelID(request.Model)
	return request, effort, nil
}
