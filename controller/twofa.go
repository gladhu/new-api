package controller

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
)

// Setup2FARequest 设置2FA请求结构
type Setup2FARequest struct {
	Code string `json:"code" binding:"required"`
}

// Enable2FARequest 启用2FA请求结构
type Enable2FARequest struct {
	Code     string `json:"code" binding:"required"`
	DeviceId int    `json:"device_id"`
}

// Verify2FARequest 验证2FA请求结构
type Verify2FARequest struct {
	Code string `json:"code" binding:"required"`
}

// Delete2FADeviceRequest 删除2FA设备请求结构
type Delete2FADeviceRequest struct {
	Code string `json:"code" binding:"required"`
}

// Setup2FAResponse 设置2FA响应结构
type Setup2FAResponse struct {
	DeviceId     int      `json:"device_id"`
	Secret       string   `json:"secret"`
	QRCodeData   string   `json:"qr_code_data"`
	BackupCodes  []string `json:"backup_codes,omitempty"`
	IsAdditional bool     `json:"is_additional"`
}

// Setup2FA 初始化2FA设置或添加新的 TOTP 设备
func Setup2FA(c *gin.Context) {
	userId := c.GetInt("id")

	existing, err := model.GetTwoFAByUserId(userId)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	isAdditional := existing != nil && existing.IsEnabled
	if isAdditional {
		total, err := model.CountTotalActiveAuthenticators(userId, existing)
		if err != nil {
			common.ApiError(c, err)
			return
		}
		if total >= common.MaxTwoFADevices {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": fmt.Sprintf("最多只能绑定 %d 个验证器", common.MaxTwoFADevices),
			})
			return
		}
		if err := model.DeletePendingTwoFADevices(userId); err != nil {
			common.ApiError(c, err)
			return
		}
	} else if existing != nil && !existing.IsEnabled {
		if err := model.DeletePendingTwoFADevices(userId); err != nil {
			common.ApiError(c, err)
			return
		}
	}

	user, err := model.GetUserById(userId, false)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	activeCount := 0
	if isAdditional {
		activeCount, err = model.CountTotalActiveAuthenticators(userId, existing)
		if err != nil {
			common.ApiError(c, err)
			return
		}
	}

	deviceIndex := activeCount + 1
	accountName := user.Username
	if deviceIndex > 1 {
		accountName = fmt.Sprintf("%s (%d)", user.Username, deviceIndex)
	}

	key, err := common.GenerateTOTPSecret(accountName)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "生成2FA密钥失败",
		})
		common.SysLog("生成TOTP密钥失败: " + err.Error())
		return
	}

	var backupCodes []string
	if !isAdditional {
		backupCodes, err = common.GenerateBackupCodes()
		if err != nil {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": "生成备用码失败",
			})
			common.SysLog("生成备用码失败: " + err.Error())
			return
		}
	}

	qrCodeData := common.GenerateQRCodeData(key.Secret(), accountName)

	var responseDeviceID int
	if isAdditional {
		deviceLabel := fmt.Sprintf("Authenticator %d", deviceIndex)
		device, err := model.CreateTwoFADevice(userId, key.Secret(), deviceLabel, true)
		if err != nil {
			common.ApiError(c, err)
			return
		}
		responseDeviceID = device.Id
	} else {
		if existing == nil {
			twoFA := &model.TwoFA{
				UserId:    userId,
				Secret:    key.Secret(),
				IsEnabled: false,
			}
			if err := twoFA.Create(); err != nil {
				common.ApiError(c, err)
				return
			}
		} else {
			existing.Secret = key.Secret()
			if err := existing.Update(); err != nil {
				common.ApiError(c, err)
				return
			}
		}

		if err := model.CreateBackupCodes(userId, backupCodes); err != nil {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": "保存备用码失败",
			})
			common.SysLog("保存备用码失败: " + err.Error())
			return
		}
		responseDeviceID = model.LegacyPrimaryDeviceID
	}

	logMessage := "开始设置两步验证"
	if isAdditional {
		logMessage = fmt.Sprintf("开始添加第 %d 个两步验证设备", deviceIndex)
	}
	model.RecordLog(userId, model.LogTypeSystem, logMessage)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "2FA设置初始化成功，请使用认证器扫描二维码并输入验证码完成设置",
		"data": Setup2FAResponse{
			DeviceId:     responseDeviceID,
			Secret:       key.Secret(),
			QRCodeData:   qrCodeData,
			BackupCodes:  backupCodes,
			IsAdditional: isAdditional,
		},
	})
}

// Enable2FA 启用2FA或确认新增设备
func Enable2FA(c *gin.Context) {
	var req Enable2FARequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "参数错误",
		})
		return
	}

	userId := c.GetInt("id")

	twoFA, err := model.GetTwoFAByUserId(userId)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if twoFA == nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "请先完成2FA初始化设置",
		})
		return
	}

	cleanCode, err := common.ValidateNumericCode(req.Code)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	if req.DeviceId <= model.LegacyPrimaryDeviceID {
		valid, err := twoFA.ValidateLegacyTOTPAndUpdateUsage(cleanCode)
		if err != nil {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": err.Error(),
			})
			return
		}
		if !valid {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": "验证码或备用码错误，请重试",
			})
			return
		}
		if err := twoFA.Enable(); err != nil {
			common.ApiError(c, err)
			return
		}
		model.RecordLog(userId, model.LogTypeSystem, "成功启用两步验证")
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"message": "两步验证启用成功",
		})
		return
	}

	device, err := model.GetTwoFADeviceById(userId, req.DeviceId)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if device == nil || !device.IsPending {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "未找到待确认的验证器，请重新发起设置",
		})
		return
	}

	valid, err := twoFA.ValidateTOTPForDevice(cleanCode, device.Id)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	if !valid {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "验证码或备用码错误，请重试",
		})
		return
	}

	if err := device.Activate(); err != nil {
		common.ApiError(c, err)
		return
	}

	model.RecordLog(userId, model.LogTypeSystem, fmt.Sprintf("成功添加两步验证设备：%s", device.Label))
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "验证器添加成功",
	})
}

// Disable2FA 禁用2FA
func Disable2FA(c *gin.Context) {
	var req Verify2FARequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "参数错误",
		})
		return
	}

	userId := c.GetInt("id")

	twoFA, err := model.GetTwoFAByUserId(userId)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if twoFA == nil || !twoFA.IsEnabled {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "用户未启用2FA",
		})
		return
	}

	if !verifyTwoFACode(twoFA, req.Code, c) {
		return
	}

	if err := model.DisableTwoFA(userId); err != nil {
		common.ApiError(c, err)
		return
	}

	model.RecordLog(userId, model.LogTypeSystem, "禁用两步验证")

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "两步验证已禁用",
	})
}

// Delete2FADevice 删除单个 TOTP 设备
func Delete2FADevice(c *gin.Context) {
	deviceId, err := strconv.Atoi(c.Param("id"))
	if err != nil || deviceId <= 0 {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "设备ID格式错误",
		})
		return
	}

	var req Delete2FADeviceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "参数错误",
		})
		return
	}

	userId := c.GetInt("id")
	twoFA, err := model.GetTwoFAByUserId(userId)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if twoFA == nil || !twoFA.IsEnabled {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "用户未启用2FA",
		})
		return
	}

	device, err := model.GetTwoFADeviceById(userId, deviceId)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if device == nil || device.IsPending {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "设备不存在",
		})
		return
	}

	activeCount, err := model.CountTotalActiveAuthenticators(userId, twoFA)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if activeCount <= 1 {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "至少保留一个验证器，如需关闭请禁用两步验证",
		})
		return
	}

	if !verifyTwoFACode(twoFA, req.Code, c) {
		return
	}

	deviceLabel := device.Label
	if err := model.DeleteTwoFADeviceById(userId, deviceId); err != nil {
		common.ApiError(c, err)
		return
	}

	model.RecordLog(userId, model.LogTypeSystem, fmt.Sprintf("删除两步验证设备：%s", deviceLabel))
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "验证器已删除",
	})
}

func verifyTwoFACode(twoFA *model.TwoFA, code string, c *gin.Context) bool {
	cleanCode, err := common.ValidateNumericCode(code)
	isValidTOTP := false
	isValidBackup := false

	if err == nil {
		isValidTOTP, _ = twoFA.ValidateTOTPAndUpdateUsage(cleanCode)
	}

	if !isValidTOTP {
		isValidBackup, err = twoFA.ValidateBackupCodeAndUpdateUsage(code)
		if err != nil {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": err.Error(),
			})
			return false
		}
	}

	if !isValidTOTP && !isValidBackup {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "验证码或备用码错误，请重试",
		})
		return false
	}

	return true
}

// Get2FAStatus 获取用户2FA状态
func Get2FAStatus(c *gin.Context) {
	userId := c.GetInt("id")

	twoFA, err := model.GetTwoFAByUserId(userId)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	status := map[string]interface{}{
		"enabled":      false,
		"locked":       false,
		"device_count": 0,
		"max_devices":  common.MaxTwoFADevices,
		"devices":      []map[string]interface{}{},
	}

	if twoFA != nil {
		status["enabled"] = twoFA.IsEnabled
		status["locked"] = twoFA.IsLocked()
		if twoFA.IsEnabled {
			backupCount, err := model.GetUnusedBackupCodeCount(userId)
			if err != nil {
				common.SysLog("获取备用码数量失败: " + err.Error())
			} else {
				status["backup_codes_remaining"] = backupCount
			}

			devices, err := model.GetEffectiveActiveTwoFADevices(twoFA)
			if err != nil {
				common.ApiError(c, err)
				return
			}

			totalCount, err := model.CountTotalActiveAuthenticators(userId, twoFA)
			if err != nil {
				common.ApiError(c, err)
				return
			}
			status["device_count"] = totalCount

			deviceList := make([]map[string]interface{}, 0, len(devices)+1)
			if model.HasLegacyPrimarySecret(twoFA) {
				primary := map[string]interface{}{
					"id":         model.LegacyPrimaryDeviceID,
					"label":      "Authenticator 1",
					"is_primary": true,
				}
				if !twoFA.CreatedAt.IsZero() {
					primary["created_at"] = twoFA.CreatedAt.Unix()
				}
				if twoFA.LastUsedAt != nil {
					primary["last_used_at"] = twoFA.LastUsedAt.Unix()
				}
				deviceList = append(deviceList, primary)
			}
			for _, device := range devices {
				item := map[string]interface{}{
					"id":         device.Id,
					"label":      device.Label,
					"is_primary": false,
				}
				if !device.CreatedAt.IsZero() {
					item["created_at"] = device.CreatedAt.Unix()
				}
				if device.LastUsedAt != nil {
					item["last_used_at"] = device.LastUsedAt.Unix()
				}
				deviceList = append(deviceList, item)
			}
			status["devices"] = deviceList
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    status,
	})
}

// RegenerateBackupCodes 重新生成备用码
func RegenerateBackupCodes(c *gin.Context) {
	var req Verify2FARequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "参数错误",
		})
		return
	}

	userId := c.GetInt("id")

	twoFA, err := model.GetTwoFAByUserId(userId)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if twoFA == nil || !twoFA.IsEnabled {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "用户未启用2FA",
		})
		return
	}

	cleanCode, err := common.ValidateNumericCode(req.Code)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	valid, err := twoFA.ValidateTOTPAndUpdateUsage(cleanCode)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	if !valid {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "验证码或备用码错误，请重试",
		})
		return
	}

	backupCodes, err := common.GenerateBackupCodes()
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "生成备用码失败",
		})
		common.SysLog("生成备用码失败: " + err.Error())
		return
	}

	if err := model.CreateBackupCodes(userId, backupCodes); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "保存备用码失败",
		})
		common.SysLog("保存备用码失败: " + err.Error())
		return
	}

	model.RecordLog(userId, model.LogTypeSystem, "重新生成两步验证备用码")

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "备用码重新生成成功",
		"data": map[string]interface{}{
			"backup_codes": backupCodes,
		},
	})
}

// Verify2FALogin 登录时验证2FA
func Verify2FALogin(c *gin.Context) {
	var req Verify2FARequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "参数错误",
		})
		return
	}

	session := sessions.Default(c)
	pendingUserId := session.Get("pending_user_id")
	if pendingUserId == nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "会话已过期，请重新登录",
		})
		return
	}
	userId, ok := pendingUserId.(int)
	if !ok {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "会话数据无效，请重新登录",
		})
		return
	}

	user, err := model.GetUserById(userId, false)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "用户不存在",
		})
		return
	}

	twoFA, err := model.GetTwoFAByUserId(user.Id)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if twoFA == nil || !twoFA.IsEnabled {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "用户未启用2FA",
		})
		return
	}

	if !verifyTwoFACode(twoFA, req.Code, c) {
		return
	}

	session.Delete("pending_username")
	session.Delete("pending_user_id")
	session.Save()

	setupLogin(user, c)
}

// Admin2FAStats 管理员获取2FA统计信息
func Admin2FAStats(c *gin.Context) {
	stats, err := model.GetTwoFAStats()
	if err != nil {
		common.ApiError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    stats,
	})
}

// AdminDisable2FA 管理员强制禁用用户2FA
func AdminDisable2FA(c *gin.Context) {
	userIdStr := c.Param("id")
	userId, err := strconv.Atoi(userIdStr)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "用户ID格式错误",
		})
		return
	}

	targetUser, err := model.GetUserById(userId, false)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	myRole := c.GetInt("role")
	if !canManageTargetRole(myRole, targetUser.Role) {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "无权操作同级或更高级用户的2FA设置",
		})
		return
	}

	if err := model.DisableTwoFA(userId); err != nil {
		if errors.Is(err, model.ErrTwoFANotEnabled) {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": "用户未启用2FA",
			})
			return
		}
		common.ApiError(c, err)
		return
	}

	adminId := c.GetInt("id")
	adminName := c.GetString("username")
	adminInfo := map[string]interface{}{
		"admin_id":       adminId,
		"admin_username": adminName,
	}
	model.RecordLogWithAdminInfo(userId, model.LogTypeManage,
		"管理员强制禁用了用户的两步验证", adminInfo)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "用户2FA已被强制禁用",
	})
}
