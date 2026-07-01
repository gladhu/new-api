package aws

import (
	"fmt"
	"io"
	"net/http"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/relay/channel"
	"github.com/QuantumNous/new-api/relay/channel/claude"
	"github.com/QuantumNous/new-api/relay/channel/openai"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/types"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
	"github.com/pkg/errors"

	"github.com/gin-gonic/gin"
)

type ClientMode int

const (
	ClientModeApiKey ClientMode = iota + 1
	ClientModeAKSK
)

type Adaptor struct {
	ClientMode      ClientMode
	AwsClient       *bedrockruntime.Client
	AwsModelId      string
	AwsReq          any
	IsNova          bool
	IsBedrockOpenAI bool
}

func (a *Adaptor) ConvertGeminiRequest(*gin.Context, *relaycommon.RelayInfo, *dto.GeminiChatRequest) (any, error) {
	//TODO implement me
	return nil, errors.New("not implemented")
}

func (a *Adaptor) ConvertClaudeRequest(c *gin.Context, info *relaycommon.RelayInfo, request *dto.ClaudeRequest) (any, error) {
	for i, message := range request.Messages {
		updated := false
		if !message.IsStringContent() {
			content, err := message.ParseContent()
			if err != nil {
				return nil, errors.Wrap(err, "failed to parse message content")
			}
			for i2, mediaMessage := range content {
				if mediaMessage.Source != nil {
					if mediaMessage.Source.Type == "url" {
						// 使用统一的文件服务获取图片数据
						source := types.NewURLFileSource(mediaMessage.Source.Url)
						base64Data, mimeType, err := service.GetBase64Data(c, source, "formatting image for Claude")
						if err != nil {
							return nil, fmt.Errorf("get file base64 from url failed: %s", err.Error())
						}
						mediaMessage.Source.MediaType = mimeType
						mediaMessage.Source.Data = base64Data
						mediaMessage.Source.Url = ""
						mediaMessage.Source.Type = "base64"
						content[i2] = mediaMessage
						updated = true
					}
				}
			}
			if updated {
				message.SetContent(content)
			}
		}
		if updated {
			request.Messages[i] = message
		}
	}
	return request, nil
}

func (a *Adaptor) ConvertAudioRequest(c *gin.Context, info *relaycommon.RelayInfo, request dto.AudioRequest) (io.Reader, error) {
	//TODO implement me
	return nil, errors.New("not implemented")
}

func (a *Adaptor) ConvertImageRequest(c *gin.Context, info *relaycommon.RelayInfo, request dto.ImageRequest) (any, error) {
	//TODO implement me
	return nil, errors.New("not implemented")
}

func (a *Adaptor) Init(info *relaycommon.RelayInfo) {
	if info == nil {
		return
	}
	a.IsBedrockOpenAI = common.IsBedrockOpenAIModel(info.UpstreamModelName) || common.IsBedrockOpenAIModel(info.OriginModelName)
}

func (a *Adaptor) GetRequestURL(info *relaycommon.RelayInfo) (string, error) {
	if a.isBedrockOpenAIRequest(info) {
		a.IsBedrockOpenAI = true
		if info.ChannelOtherSettings.AwsKeyType != dto.AwsKeyTypeApiKey {
			return "", errors.New("Bedrock OpenAI models require API Key authentication (aws_key_type: api_key)")
		}
		if info.RelayMode != relayconstant.RelayModeResponses {
			return "", errors.New("Bedrock OpenAI models only support /v1/responses")
		}
		a.ClientMode = ClientModeApiKey
		_, region, err := parseAwsApiKeyAndRegion(info.ApiKey)
		if err != nil {
			return "", err
		}
		return bedrockMantleResponsesURL(region), nil
	}

	if info.ChannelOtherSettings.AwsKeyType == dto.AwsKeyTypeApiKey {
		awsModelId := getAwsModelID(info.UpstreamModelName)
		a.ClientMode = ClientModeApiKey
		_, region, err := parseAwsApiKeyAndRegion(info.ApiKey)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("https://bedrock-runtime.%s.amazonaws.com/model/%s/converse", region, awsModelId), nil
	}

	a.ClientMode = ClientModeAKSK
	return "", nil
}

func (a *Adaptor) SetupRequestHeader(c *gin.Context, req *http.Header, info *relaycommon.RelayInfo) error {
	if a.IsBedrockOpenAI {
		channel.SetupApiRequestHeader(info, c, req)
		token, _, err := parseAwsApiKeyAndRegion(info.ApiKey)
		if err != nil {
			return err
		}
		req.Set("Authorization", "Bearer "+token)
		return nil
	}

	claude.CommonClaudeHeadersOperation(c, req, info)
	if a.ClientMode == ClientModeApiKey {
		token, _, err := parseAwsApiKeyAndRegion(info.ApiKey)
		if err != nil {
			return err
		}
		req.Set("Authorization", "Bearer "+token)
	}
	return nil
}

func (a *Adaptor) ConvertOpenAIRequest(c *gin.Context, info *relaycommon.RelayInfo, request *dto.GeneralOpenAIRequest) (any, error) {
	if request == nil {
		return nil, errors.New("request is nil")
	}
	if common.IsBedrockOpenAIModel(request.Model) || common.IsBedrockOpenAIModel(info.UpstreamModelName) {
		return nil, errors.New("Bedrock OpenAI models only support /v1/responses")
	}
	// 检查是否为Nova模型
	if isNovaModel(request.Model) {
		novaReq := convertToNovaRequest(request)
		a.IsNova = true
		return novaReq, nil
	}

	// 原有的Claude模型处理逻辑
	claudeReq, err := claude.RequestOpenAI2ClaudeMessage(c, *request)
	if err != nil {
		return nil, errors.Wrap(err, "failed to convert openai request to claude request")
	}
	info.UpstreamModelName = claudeReq.Model
	return claudeReq, err
}

func (a *Adaptor) ConvertRerankRequest(c *gin.Context, relayMode int, request dto.RerankRequest) (any, error) {
	return nil, nil
}

func (a *Adaptor) ConvertEmbeddingRequest(c *gin.Context, info *relaycommon.RelayInfo, request dto.EmbeddingRequest) (any, error) {
	//TODO implement me
	return nil, errors.New("not implemented")
}

func (a *Adaptor) ConvertOpenAIResponsesRequest(c *gin.Context, info *relaycommon.RelayInfo, request dto.OpenAIResponsesRequest) (any, error) {
	converted, effort, err := convertBedrockOpenAIResponsesRequest(info.UpstreamModelName, request)
	if err != nil {
		return nil, err
	}
	info.UpstreamModelName = converted.Model
	if effort != "" {
		info.ReasoningEffort = effort
	} else if converted.Reasoning != nil && converted.Reasoning.Effort != "" {
		info.ReasoningEffort = converted.Reasoning.Effort
	}
	a.IsBedrockOpenAI = true
	return converted, nil
}

func (a *Adaptor) DoRequest(c *gin.Context, info *relaycommon.RelayInfo, requestBody io.Reader) (any, error) {
	if a.IsBedrockOpenAI || a.isBedrockOpenAIRequest(info) {
		return channel.DoApiRequest(a, c, info, requestBody)
	}
	if a.ClientMode == ClientModeApiKey {
		return channel.DoApiRequest(a, c, info, requestBody)
	}
	return doAwsClientRequest(c, info, a, requestBody)
}

func (a *Adaptor) DoResponse(c *gin.Context, resp *http.Response, info *relaycommon.RelayInfo) (usage any, err *types.NewAPIError) {
	if a.IsBedrockOpenAI || a.isBedrockOpenAIRequest(info) {
		if info.IsStream {
			return openai.OaiResponsesStreamHandler(c, info, resp)
		}
		return openai.OaiResponsesHandler(c, info, resp)
	}

	if a.ClientMode == ClientModeApiKey {
		claudeAdaptor := claude.Adaptor{}
		usage, err = claudeAdaptor.DoResponse(c, resp, info)
	} else {
		if a.IsNova {
			err, usage = handleNovaRequest(c, info, a)
		} else {
			if info.IsStream {
				err, usage = awsStreamHandler(c, info, a)
			} else {
				err, usage = awsHandler(c, info, a)
			}
		}
	}
	return
}

func (a *Adaptor) GetModelList() (models []string) {
	for n := range awsModelIDMap {
		models = append(models, n)
	}

	return
}

func (a *Adaptor) GetChannelName() string {
	return ChannelName
}

func (a *Adaptor) isBedrockOpenAIRequest(info *relaycommon.RelayInfo) bool {
	if info == nil {
		return false
	}
	return common.IsBedrockOpenAIModel(info.UpstreamModelName) || common.IsBedrockOpenAIModel(info.OriginModelName)
}
