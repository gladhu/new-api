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
			if outboundStorage != nil {
				rawBody, readErr := outboundStorage.Bytes()
				for readErr == nil && httpResp != nil && httpResp.StatusCode != http.StatusOK && newAPIError != nil {
					var (
						strippedBody []byte
						stripped     bool
						retryReason  string
					)
					switch {
					case isEncryptedReasoningError(httpResp.StatusCode, newAPIError):
						strippedBody, stripped = stripEncryptedReasoningFromResponsesBody(rawBody)
						if stripped {
							stripEncryptedReasoningFromRelayRequest(info.Request)
							retryReason = "rejected encrypted reasoning content, retrying after stripping encrypted reasoning from input"
						}
					case isUnsupportedSearchContentTypesError(httpResp.StatusCode, newAPIError):
						strippedBody, stripped = stripSearchContentTypesFromResponsesBody(rawBody)
						if stripped {
							stripSearchContentTypesFromRelayRequest(info.Request)
							retryReason = "rejected search_content_types in web search tool, retrying after stripping search_content_types"
						}
					}
					if !stripped {
						break
					}
					rawBody = strippedBody
					retryBody, retryCloser, bodyErr := relaycommon.NewOutboundJSONBody(rawBody)
					if bodyErr != nil {
						break
					}
					defer retryCloser.Close()
					logger.LogInfo(c, fmt.Sprintf("channel #%d %s", info.ChannelId, retryReason))
					retryResp, retryErr := adaptor.DoRequest(c, info, retryBody)
					if retryErr != nil {
						return types.NewOpenAIError(retryErr, types.ErrorCodeDoRequestFailed, http.StatusInternalServerError)
					}
					if retryResp == nil {
						break
					}
					httpResp = retryResp.(*http.Response)
					if httpResp.StatusCode == http.StatusOK {
						newAPIError = nil
						break
					}
					newAPIError = service.RelayErrorHandler(c.Request.Context(), httpResp, false)
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
	if code == "invalid_encrypted_content" || code == "invalid_encrypted_reasoning" {
		return true
	}
	msg := strings.ToLower(err.Error())
	// Bedrock often returns a bare message with no invalid_encrypted_content
	// code (or only a validation_error code): "invalid encrypted content",
	// "invalid encrypted reasoning", or "invalid encrypted view".
	if strings.Contains(msg, "invalid encrypted content") ||
		strings.Contains(msg, "invalid encrypted reasoning") ||
		strings.Contains(msg, "invalid encrypted view") ||
		strings.Contains(msg, "encrypted reasoning") ||
		strings.Contains(msg, "encrypted content could not be decrypted") ||
		strings.Contains(msg, "encrypted content could not be verified") {
		return true
	}
	if (strings.Contains(msg, "encrypted content") || strings.Contains(msg, "encrypted_content")) &&
		(strings.Contains(msg, "could not be verified") ||
			strings.Contains(msg, "could not be decrypted") ||
			strings.Contains(msg, "could not be parsed") ||
			strings.Contains(msg, "different account") ||
			strings.Contains(msg, "different provider") ||
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
	kept, changed := filterEncryptedInputItems(items)
	if !changed || len(kept) == 0 {
		return raw, false
	}
	cleaned, err := common.Marshal(kept)
	if err != nil {
		return raw, false
	}
	return cleaned, true
}

func filterEncryptedInputItems(items []any) ([]any, bool) {
	kept := make([]any, 0, len(items))
	changed := false
	for _, item := range items {
		cleaned, itemChanged, drop := sanitizeEncryptedInputItem(item)
		if itemChanged {
			changed = true
		}
		if drop {
			continue
		}
		kept = append(kept, cleaned)
	}
	return kept, changed
}

func sanitizeEncryptedInputItem(item any) (any, bool, bool) {
	m, ok := item.(map[string]any)
	if !ok {
		cleaned, childChanged := stripEncryptedValue(item)
		if cleaned == nil {
			return nil, true, true
		}
		return cleaned, childChanged, false
	}
	typ, _ := m["type"].(string)
	if isReasoningItemType(typ) {
		return nil, true, true
	}
	_, hadEncrypted := m["encrypted_content"]
	cleaned, childChanged := stripEncryptedMap(m)
	// Compaction checkpoints are the ciphertext itself. Leaving the shell
	// makes Bedrock reject the retry with the same invalid encrypted content.
	if isCompactionItemType(typ) && (hadEncrypted || isOpaqueEncryptedShell(cleaned)) {
		return nil, true, true
	}
	if hadEncrypted && isOpaqueEncryptedShell(cleaned) {
		return nil, true, true
	}
	return cleaned, hadEncrypted || childChanged, false
}

func stripEncryptedValue(v any) (any, bool) {
	switch val := v.(type) {
	case map[string]any:
		typ, _ := val["type"].(string)
		if isReasoningItemType(typ) {
			return nil, true
		}
		cleaned, changed := stripEncryptedMap(val)
		if isCompactionItemType(typ) && isOpaqueEncryptedShell(cleaned) {
			return nil, true
		}
		return cleaned, changed
	case []any:
		filtered, changed := filterEncryptedInputItems(val)
		if !changed {
			return val, false
		}
		return filtered, true
	default:
		return v, false
	}
}

func stripEncryptedMap(m map[string]any) (map[string]any, bool) {
	changed := false
	if _, ok := m["encrypted_content"]; ok {
		delete(m, "encrypted_content")
		changed = true
	}
	for k, v := range m {
		cleaned, childChanged := stripEncryptedValue(v)
		if !childChanged {
			continue
		}
		changed = true
		if cleaned == nil {
			delete(m, k)
			continue
		}
		m[k] = cleaned
	}
	return m, changed
}

func isReasoningItemType(typ string) bool {
	return strings.EqualFold(strings.TrimSpace(typ), "reasoning")
}

func isCompactionItemType(typ string) bool {
	return strings.EqualFold(strings.TrimSpace(typ), "compaction")
}

// isOpaqueEncryptedShell reports an item that has no readable conversation
// payload left after encrypted reasoning fields are removed.
func isOpaqueEncryptedShell(item map[string]any) bool {
	if hasResponsesItemPayload(item) {
		return false
	}
	for k, v := range item {
		switch k {
		case "id", "type", "status", "encrypted_content":
			continue
		}
		if v != nil {
			return false
		}
	}
	return true
}

func hasResponsesItemPayload(item map[string]any) bool {
	for _, key := range []string{"role", "content", "call_id", "output", "arguments", "input", "name", "summary", "text"} {
		if _, ok := item[key]; ok {
			return true
		}
	}
	return false
}

func isUnsupportedSearchContentTypesError(statusCode int, err *types.NewAPIError) bool {
	if statusCode != http.StatusBadRequest || err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	if !strings.Contains(msg, "search_content_types") {
		return false
	}
	return strings.Contains(msg, "not supported") ||
		strings.Contains(msg, "unsupported") ||
		strings.Contains(msg, "unknown") ||
		strings.Contains(msg, "unrecognized") ||
		strings.Contains(msg, "not permitted") ||
		strings.Contains(msg, "not allowed")
}

func stripSearchContentTypesFromRelayRequest(request any) {
	switch req := request.(type) {
	case *dto.OpenAIResponsesRequest:
		if cleaned, ok := stripSearchContentTypesFromTools(req.Tools); ok {
			req.Tools = cleaned
		}
		if cleaned, ok := stripSearchContentTypesFromInputTools(req.Input); ok {
			req.Input = cleaned
		}
	case *dto.OpenAIResponsesCompactionRequest:
		if cleaned, ok := stripSearchContentTypesFromTools(req.Tools); ok {
			req.Tools = cleaned
		}
		if cleaned, ok := stripSearchContentTypesFromInputTools(req.Input); ok {
			req.Input = cleaned
		}
	}
}

func stripSearchContentTypesFromResponsesBody(body []byte) ([]byte, bool) {
	if len(body) == 0 {
		return nil, false
	}
	updatedBody := body
	changed := false

	toolsResult := gjson.GetBytes(updatedBody, "tools")
	if toolsResult.Exists() && toolsResult.IsArray() {
		if cleanedTools, ok := stripSearchContentTypesFromTools(json.RawMessage(toolsResult.Raw)); ok {
			nextBody, err := sjson.SetRawBytes(updatedBody, "tools", cleanedTools)
			if err == nil {
				updatedBody = nextBody
				changed = true
			}
		}
	}

	inputResult := gjson.GetBytes(updatedBody, "input")
	if inputResult.Exists() && inputResult.IsArray() {
		if cleanedInput, ok := stripSearchContentTypesFromInputTools(json.RawMessage(inputResult.Raw)); ok {
			nextBody, err := sjson.SetRawBytes(updatedBody, "input", cleanedInput)
			if err == nil {
				updatedBody = nextBody
				changed = true
			}
		}
	}

	if !changed {
		return nil, false
	}
	return updatedBody, true
}

func stripSearchContentTypesFromTools(raw json.RawMessage) (json.RawMessage, bool) {
	if len(raw) == 0 || common.GetJsonType(raw) != "array" {
		return raw, false
	}
	var tools []any
	if err := common.Unmarshal(raw, &tools); err != nil {
		return raw, false
	}
	if !stripSearchContentTypesFromToolSlice(tools) {
		return raw, false
	}
	cleaned, err := common.Marshal(tools)
	if err != nil {
		return raw, false
	}
	return cleaned, true
}

func stripSearchContentTypesFromInputTools(raw json.RawMessage) (json.RawMessage, bool) {
	if len(raw) == 0 || common.GetJsonType(raw) != "array" {
		return raw, false
	}
	var items []any
	if err := common.Unmarshal(raw, &items); err != nil {
		return raw, false
	}
	changed := false
	for _, item := range items {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		tools, ok := m["tools"].([]any)
		if !ok {
			continue
		}
		if stripSearchContentTypesFromToolSlice(tools) {
			changed = true
		}
	}
	if !changed {
		return raw, false
	}
	cleaned, err := common.Marshal(items)
	if err != nil {
		return raw, false
	}
	return cleaned, true
}

func stripSearchContentTypesFromToolSlice(tools []any) bool {
	changed := false
	for _, tool := range tools {
		m, ok := tool.(map[string]any)
		if !ok {
			continue
		}
		if _, exists := m["search_content_types"]; exists {
			delete(m, "search_content_types")
			changed = true
		}
	}
	return changed
}
