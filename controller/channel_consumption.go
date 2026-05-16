package controller

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"

	"github.com/gin-gonic/gin"
)

func parseChannelConsumptionTimeRange(c *gin.Context) (startTimestamp, endTimestamp int64, err error) {
	startTimestamp, _ = strconv.ParseInt(c.Query("start_timestamp"), 10, 64)
	endTimestamp, _ = strconv.ParseInt(c.Query("end_timestamp"), 10, 64)
	if startTimestamp == 0 && endTimestamp == 0 {
		now := time.Now()
		loc := now.Location()
		monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, loc)
		startTimestamp = monthStart.Unix()
		endTimestamp = now.Unix()
		return startTimestamp, endTimestamp, nil
	}
	if startTimestamp == 0 || endTimestamp == 0 {
		return 0, 0, errors.New("start_timestamp and end_timestamp are required")
	}
	if endTimestamp < startTimestamp {
		return 0, 0, errors.New("end_timestamp must be greater than or equal to start_timestamp")
	}
	return startTimestamp, endTimestamp, nil
}

// GetChannelConsumption returns consume quota aggregated for a channel in [start_timestamp, end_timestamp].
// Optional filters: user_id or username (for per-user consumption on this channel).
func GetChannelConsumption(c *gin.Context) {
	channelId, err := strconv.Atoi(c.Param("id"))
	if err != nil || channelId <= 0 {
		common.ApiError(c, errors.New("invalid channel id"))
		return
	}

	channel, err := model.GetChannelById(channelId, false)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	startTimestamp, endTimestamp, err := parseChannelConsumptionTimeRange(c)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	userId, _ := strconv.Atoi(c.Query("user_id"))
	username := strings.TrimSpace(c.Query("username"))
	if userId > 0 {
		if _, err := model.GetUserById(userId, false); err != nil {
			common.ApiError(c, errors.New("user not found"))
			return
		}
		username = ""
	} else if username != "" {
		if _, err := model.GetUserByUsername(username); err != nil {
			common.ApiError(c, errors.New("user not found"))
			return
		}
	}

	stat, err := model.SumChannelConsumption(channelId, userId, username, startTimestamp, endTimestamp)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	resp := gin.H{
		"channel_id":          channel.Id,
		"channel_name":        channel.Name,
		"start_timestamp":     startTimestamp,
		"end_timestamp":       endTimestamp,
		"quota":               stat.Quota,
		"request_count":       stat.RequestCount,
		"prompt_tokens":       stat.PromptTokens,
		"completion_tokens":   stat.CompletionTokens,
		"lifetime_used_quota": channel.UsedQuota,
	}
	if userId > 0 {
		resp["user_id"] = userId
		if user, uerr := model.GetUserById(userId, false); uerr == nil && user != nil {
			resp["username"] = user.Username
		}
	} else if username != "" {
		resp["username"] = username
		if user, uerr := model.GetUserByUsername(username); uerr == nil && user != nil {
			resp["user_id"] = user.Id
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    resp,
	})
}
