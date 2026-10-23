package relay

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/QuantumNous/new-api/relay/helper"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/QuantumNous/new-api/relaykit/types"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/model_setting"

	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

func ResponsesHelper(c *gin.Context, info *relaycommon.RelayInfo) (newAPIError *types.NewAPIError) {
	info.InitChannelMeta(c)
	if info.RelayMode == relayconstant.RelayModeResponsesCompact &&
		!common.SupportsResponsesCompact(info.ChannelType, info.ApiType) {
		return types.NewErrorWithStatusCode(
			fmt.Errorf("unsupported endpoint %q for api type %d", "/v1/responses/compact", info.ApiType),
			types.ErrorCodeInvalidRequest,
			http.StatusBadRequest,
			types.ErrOptionWithSkipRetry(),
		)
	}

	var responsesReq *dto.OpenAIResponsesRequest
	switch req := info.Request.(type) {
	case *dto.OpenAIResponsesRequest:
		responsesReq = req
	case *dto.OpenAIResponsesCompactionRequest:
		// Only fields documented for POST /v1/responses/compact are forwarded:
		// model, input, instructions, previous_response_id, prompt_cache_key,
		// prompt_cache_options, prompt_cache_retention, service_tier.
		// Undocumented Codex-parity fields (tools, reasoning, text) are parsed
		// for client compatibility but intentionally not sent upstream.
		responsesReq = &dto.OpenAIResponsesRequest{
			Model:                req.Model,
			Input:                req.Input,
			Instructions:         req.Instructions,
			PreviousResponseID:   req.PreviousResponseID,
			ParallelToolCalls:    req.ParallelToolCalls,
			ServiceTier:          req.ServiceTier,
			PromptCacheKey:       req.PromptCacheKey,
			PromptCacheOptions:   req.PromptCacheOptions,
			PromptCacheRetention: req.PromptCacheRetention,
		}
	default:
		return types.NewErrorWithStatusCode(
			fmt.Errorf("invalid request type, expected dto.OpenAIResponsesRequest or dto.OpenAIResponsesCompactionRequest, got %T", info.Request),
			types.ErrorCodeInvalidRequest,
			http.StatusBadRequest,
			types.ErrOptionWithSkipRetry(),
		)
	}

	request, err := common.DeepCopy(responsesReq)
	if err != nil {
		return types.NewError(fmt.Errorf("failed to copy request to GeneralOpenAIRequest: %w", err), types.ErrorCodeInvalidRequest, types.ErrOptionWithSkipRetry())
	}

	err = helper.ModelMappedHelper(c, info, request)
	if err != nil {
		return types.NewError(err, types.ErrorCodeChannelModelMappedError, types.ErrOptionWithSkipRetry())
	}

	adaptor := GetAdaptor(info.ApiType)
	if adaptor == nil {
		return types.NewError(fmt.Errorf("invalid api type: %d", info.ApiType), types.ErrorCodeInvalidApiType, types.ErrOptionWithSkipRetry())
	}
	adaptor.Init(info)
	var requestBody io.Reader
	var outboundStorage common.BodyStorage
	if model_setting.GetGlobalSettings().PassThroughRequestEnabled || info.ChannelSetting.PassThroughBodyEnabled {
		storage, err := common.GetBodyStorage(c)
		if err != nil {
			return types.NewError(err, types.ErrorCodeReadRequestBodyFailed, types.ErrOptionWithSkipRetry())
		}
		outboundStorage = storage
		requestBody = common.NewReplayableBodyReader(storage)
	} else {
		convertedRequest, err := adaptor.ConvertOpenAIResponsesRequest(c, info, *request)
		if err != nil {
			return types.NewError(err, types.ErrorCodeConvertRequestFailed, types.ErrOptionWithSkipRetry())
		}
		relaycommon.AppendRequestConversionFromRequest(info, convertedRequest)
		jsonData, err := common.Marshal(convertedRequest)
		if err != nil {
			return types.NewError(err, types.ErrorCodeConvertRequestFailed, types.ErrOptionWithSkipRetry())
		}

		// remove disabled fields for OpenAI Responses API
		jsonData, err = relaycommon.RemoveDisabledFields(jsonData, info.ChannelOtherSettings, info.ChannelSetting.PassThroughBodyEnabled)
		if err != nil {
			return types.NewError(err, types.ErrorCodeConvertRequestFailed, types.ErrOptionWithSkipRetry())
		}

		// apply param override
		if len(info.ParamOverride) > 0 {
			jsonData, err = relaycommon.ApplyParamOverrideWithRelayInfo(jsonData, info)
			if err != nil {
				return newAPIErrorFromParamOverride(err)
			}
		}

		logger.LogDebug(c, "requestBody: %s", jsonData)
		body, closer, err := relaycommon.NewOutboundJSONBody(jsonData)
		if err != nil {
			return types.NewError(err, types.ErrorCodeConvertRequestFailed, types.ErrOptionWithSkipRetry())
		}
		defer closer.Close()
		if bs, ok := closer.(common.BodyStorage); ok {
			outboundStorage = bs
		}
		jsonData = nil
		requestBody = body
	}

	var httpResp *http.Response
	resp, err := adaptor.DoRequest(c, info, requestBody)
	if err != nil {
		return types.NewOpenAIError(err, types.ErrorCodeDoRequestFailed, http.StatusInternalServerError)
	}

	statusCodeMappingStr := c.GetString("status_code_mapping")

	if resp != nil {
		httpResp = resp.(*http.Response)

		if httpResp.StatusCode != http.StatusOK {
			newAPIError = service.RelayErrorHandler(c.Request.Context(), httpResp, false)
			if isEncryptedReasoningError(httpResp.StatusCode, newAPIError) && outboundStorage != nil {
				if rawBody, readErr := outboundStorage.Bytes(); readErr == nil {
					if strippedBody, stripped := stripEncryptedReasoningFromResponsesBody(rawBody); stripped {
						stripEncryptedReasoningFromRelayRequest(info.Request)
						retryBody, retryCloser, bodyErr := relaycommon.NewOutboundJSONBody(strippedBody)
						if bodyErr == nil {
							defer retryCloser.Close()
							logger.LogInfo(c, fmt.Sprintf("channel #%d rejected encrypted reasoning content, retrying after stripping encrypted reasoning from input", info.ChannelId))
							retryResp, retryErr := adaptor.DoRequest(c, info, retryBody)
							if retryErr != nil {
								return types.NewOpenAIError(retryErr, types.ErrorCodeDoRequestFailed, http.StatusInternalServerError)
							}
							if retryResp != nil {
								httpResp = retryResp.(*http.Response)
								if httpResp.StatusCode != http.StatusOK {
									newAPIError = service.RelayErrorHandler(c.Request.Context(), httpResp, false)
									service.ResetStatusCode(newAPIError, statusCodeMappingStr)
									return newAPIError
								}
								newAPIError = nil
							}
						}
					}
				}
			}
			if newAPIError != nil {
				// reset status code 重置状态码
				service.ResetStatusCode(newAPIError, statusCodeMappingStr)
				return newAPIError
			}
		}
	}

	usage, newAPIError := adaptor.DoResponse(c, httpResp, info)
	if newAPIError != nil {
		// reset status code 重置状态码
		service.ResetStatusCode(newAPIError, statusCodeMappingStr)
		return newAPIError
	}

	usageDto := usage.(*dto.Usage)
	if info.RelayMode == relayconstant.RelayModeResponsesCompact {
		originModelName := info.OriginModelName
		originPriceData := info.PriceData

		_, err := helper.ModelPriceHelper(c, info, info.GetEstimatePromptTokens(), &types.TokenCountMeta{})
		if err != nil {
			info.OriginModelName = originModelName
			info.PriceData = originPriceData
			return types.NewError(err, types.ErrorCodeModelPriceError, types.ErrOptionWithSkipRetry(), types.ErrOptionWithStatusCode(http.StatusBadRequest))
		}
		service.PostTextConsumeQuota(c, info, usageDto, nil)

		info.OriginModelName = originModelName
		info.PriceData = originPriceData
		return nil
	}

	if strings.HasPrefix(info.OriginModelName, "gpt-4o-audio") {
		service.PostAudioConsumeQuota(c, info, usageDto, "")
	} else {
		service.PostTextConsumeQuota(c, info, usageDto, nil)
	}
	return nil
}

func isEncryptedReasoningError(statusCode int, err *types.NewAPIError) bool {
	if statusCode != http.StatusBadRequest || err == nil {
		return false
	}
	code := strings.ToLower(strings.TrimSpace(string(err.GetErrorCode())))
	if code == "invalid_encrypted_content" {
		return true
	}
	msg := strings.ToLower(err.Error())
	if strings.Contains(msg, "invalid encrypted reasoning") ||
		strings.Contains(msg, "encrypted content could not be decrypted") ||
		strings.Contains(msg, "encrypted content could not be verified") {
		return true
	}
	if (strings.Contains(msg, "encrypted content") || strings.Contains(msg, "encrypted_content")) &&
		(strings.Contains(msg, "could not be verified") ||
			strings.Contains(msg, "could not be decrypted") ||
			strings.Contains(msg, "decrypt")) {
		return true
	}
	return false
}

func stripEncryptedReasoningFromRelayRequest(request any) {
	switch req := request.(type) {
	case *dto.OpenAIResponsesRequest:
		if cleaned, ok := stripEncryptedReasoningFromInput(req.Input); ok {
			req.Input = cleaned
		}
	case *dto.OpenAIResponsesCompactionRequest:
		if cleaned, ok := stripEncryptedReasoningFromInput(req.Input); ok {
			req.Input = cleaned
		}
	}
}

func stripEncryptedReasoningFromResponsesBody(body []byte) ([]byte, bool) {
	if len(body) == 0 {
		return nil, false
	}
	inputResult := gjson.GetBytes(body, "input")
	if !inputResult.Exists() || !inputResult.IsArray() {
		return nil, false
	}
	cleanedInput, changed := stripEncryptedReasoningFromInput(json.RawMessage(inputResult.Raw))
	if !changed {
		return nil, false
	}
	updatedBody, err := sjson.SetRawBytes(body, "input", cleanedInput)
	if err != nil {
		return nil, false
	}
	return updatedBody, true
}

func stripEncryptedReasoningFromInput(raw json.RawMessage) (json.RawMessage, bool) {
	if len(raw) == 0 || common.GetJsonType(raw) != "array" {
		return raw, false
	}
	var items []any
	if err := common.Unmarshal(raw, &items); err != nil {
		return raw, false
	}
	kept := make([]any, 0, len(items))
	changed := false
	for _, item := range items {
		m, ok := item.(map[string]any)
		if !ok {
			kept = append(kept, item)
			continue
		}
		typ, _ := m["type"].(string)
		if strings.EqualFold(strings.TrimSpace(typ), "reasoning") {
			changed = true
			continue
		}
		if _, hasEncrypted := m["encrypted_content"]; hasEncrypted {
			changed = true
			if hasResponsesItemPayload(m) {
				cloned := make(map[string]any, len(m)-1)
				for k, v := range m {
					if k != "encrypted_content" {
						cloned[k] = v
					}
				}
				kept = append(kept, cloned)
			}
			continue
		}
		kept = append(kept, item)
	}
	if !changed || len(kept) == 0 {
		return raw, false
	}
	cleaned, err := common.Marshal(kept)
	if err != nil {
		return raw, false
	}
	return cleaned, true
}

func hasResponsesItemPayload(item map[string]any) bool {
	for _, key := range []string{"role", "content", "call_id", "output", "arguments", "input"} {
		if _, ok := item[key]; ok {
			return true
		}
	}
	return false
}
