package model

import (
	"errors"
	"fmt"
	"time"

	"github.com/QuantumNous/new-api/common"

	"gorm.io/gorm"
)

// TwoFA 用户2FA设置表（账户级：启用状态、锁定等）
type TwoFA struct {
	Id             int            `json:"id" gorm:"primaryKey"`
	UserId         int            `json:"user_id" gorm:"unique;not null;index"`
	Secret         string         `json:"-" gorm:"type:varchar(255);not null"` // 主验证器 TOTP 密钥（与旧版兼容）
	IsEnabled      bool           `json:"is_enabled"`
	FailedAttempts int            `json:"failed_attempts" gorm:"default:0"`
	LockedUntil    *time.Time     `json:"locked_until,omitempty"`
	LastUsedAt     *time.Time     `json:"last_used_at,omitempty"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	DeletedAt      gorm.DeletedAt `json:"-" gorm:"index"`
}

// TwoFADevice 额外绑定的 TOTP 验证器（第 2、3 个；主验证器仍保存在 TwoFA.Secret）
type TwoFADevice struct {
	Id         int            `json:"id" gorm:"primaryKey"`
	UserId     int            `json:"user_id" gorm:"not null;index"`
	Secret     string         `json:"-" gorm:"type:varchar(255);not null"`
	Label      string         `json:"label" gorm:"type:varchar(64);default:''"`
	IsPending  bool           `json:"is_pending" gorm:"default:false"`
	LastUsedAt *time.Time     `json:"last_used_at,omitempty"`
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
	DeletedAt  gorm.DeletedAt `json:"-" gorm:"index"`
}

// TwoFABackupCode 备用码使用记录表
type TwoFABackupCode struct {
	Id        int            `json:"id" gorm:"primaryKey"`
	UserId    int            `json:"user_id" gorm:"not null;index"`
	CodeHash  string         `json:"-" gorm:"type:varchar(255);not null"` // 备用码哈希
	IsUsed    bool           `json:"is_used"`
	UsedAt    *time.Time     `json:"used_at,omitempty"`
	CreatedAt time.Time      `json:"created_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}

// GetTwoFAByUserId 根据用户ID获取2FA设置
func GetTwoFAByUserId(userId int) (*TwoFA, error) {
	if userId == 0 {
		return nil, errors.New("用户ID不能为空")
	}

	var twoFA TwoFA
	err := DB.Where("user_id = ?", userId).First(&twoFA).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil // 返回nil表示未设置2FA
		}
		return nil, err
	}

	return &twoFA, nil
}

// IsTwoFAEnabled 检查用户是否启用了2FA
func IsTwoFAEnabled(userId int) (bool, error) {
	twoFA, err := GetTwoFAByUserId(userId)
	if err != nil {
		return false, err
	}
	return twoFA != nil && twoFA.IsEnabled, nil
}

// LegacyPrimaryDeviceID 主验证器在 API 中的虚拟设备 ID（实际存储在 TwoFA.Secret）
const LegacyPrimaryDeviceID = 0

// HasLegacyPrimarySecret 是否已配置主验证器密钥
func HasLegacyPrimarySecret(twoFA *TwoFA) bool {
	return twoFA != nil && twoFA.Secret != ""
}

// CountTotalActiveAuthenticators 统计已启用的验证器总数（主验证器 + 额外设备）
func CountTotalActiveAuthenticators(userId int, twoFA *TwoFA) (int, error) {
	extraDevices, err := GetEffectiveActiveTwoFADevices(twoFA)
	if err != nil {
		return 0, err
	}
	total := len(extraDevices)
	if twoFA != nil && twoFA.IsEnabled && HasLegacyPrimarySecret(twoFA) {
		total++
	}
	return total, nil
}

// GetActiveTwoFADevicesByUserId 获取用户已启用的额外 TOTP 设备
func GetActiveTwoFADevicesByUserId(userId int) ([]TwoFADevice, error) {
	var devices []TwoFADevice
	err := DB.Where("user_id = ? AND is_pending = ?", userId, false).
		Order("id ASC").
		Find(&devices).Error
	return devices, err
}

// GetEffectiveActiveTwoFADevices 获取额外设备，并排除与主验证器密钥重复的记录（兼容历史迁移数据）
func GetEffectiveActiveTwoFADevices(twoFA *TwoFA) ([]TwoFADevice, error) {
	devices, err := GetActiveTwoFADevicesByUserId(twoFA.UserId)
	if err != nil {
		return nil, err
	}
	if !HasLegacyPrimarySecret(twoFA) {
		return devices, nil
	}

	filtered := make([]TwoFADevice, 0, len(devices))
	for _, device := range devices {
		if device.Secret == twoFA.Secret {
			continue
		}
		filtered = append(filtered, device)
	}
	return filtered, nil
}

// GetTwoFADeviceById 根据 ID 获取设备
func GetTwoFADeviceById(userId, deviceId int) (*TwoFADevice, error) {
	if deviceId == 0 {
		return nil, errors.New("设备ID不能为空")
	}

	var device TwoFADevice
	err := DB.Where("id = ? AND user_id = ?", deviceId, userId).First(&device).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &device, nil
}

// CountActiveTwoFADevices 统计用户已启用的 TOTP 设备数量
func CountActiveTwoFADevices(userId int) (int, error) {
	var count int64
	err := DB.Model(&TwoFADevice{}).
		Where("user_id = ? AND is_pending = ?", userId, false).
		Count(&count).Error
	return int(count), err
}

// DeletePendingTwoFADevices 删除用户所有待确认的设备
func DeletePendingTwoFADevices(userId int) error {
	return DB.Unscoped().Where("user_id = ? AND is_pending = ?", userId, true).Delete(&TwoFADevice{}).Error
}

// CreateTwoFADevice 创建 TOTP 设备
func CreateTwoFADevice(userId int, secret, label string, isPending bool) (*TwoFADevice, error) {
	device := &TwoFADevice{
		UserId:    userId,
		Secret:    secret,
		Label:     label,
		IsPending: isPending,
	}
	if err := DB.Create(device).Error; err != nil {
		return nil, err
	}
	return device, nil
}

// ActivateTwoFADevice 将待确认设备标记为已启用
func (d *TwoFADevice) Activate() error {
	if d.Id == 0 {
		return errors.New("设备ID不能为空")
	}
	d.IsPending = false
	return DB.Save(d).Error
}

// DeleteTwoFADeviceById 删除指定设备
func DeleteTwoFADeviceById(userId, deviceId int) error {
	result := DB.Unscoped().Where("id = ? AND user_id = ?", deviceId, userId).Delete(&TwoFADevice{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("设备不存在")
	}
	return nil
}

// DeleteAllTwoFADevicesByUserId 删除用户全部额外 TOTP 设备
func DeleteAllTwoFADevicesByUserId(userId int) error {
	return DB.Unscoped().Where("user_id = ?", userId).Delete(&TwoFADevice{}).Error
}

// CreateTwoFA 创建2FA设置
func (t *TwoFA) Create() error {
	// 检查用户是否已存在2FA设置
	existing, err := GetTwoFAByUserId(t.UserId)
	if err != nil {
		return err
	}
	if existing != nil {
		return errors.New("用户已存在2FA设置")
	}

	// 验证用户存在
	var user User
	if err := DB.First(&user, t.UserId).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("用户不存在")
		}
		return err
	}

	return DB.Create(t).Error
}

// Update 更新2FA设置
func (t *TwoFA) Update() error {
	if t.Id == 0 {
		return errors.New("2FA记录ID不能为空")
	}
	return DB.Save(t).Error
}

// Delete 删除2FA设置
func (t *TwoFA) Delete() error {
	if t.Id == 0 {
		return errors.New("2FA记录ID不能为空")
	}

	// 使用事务确保原子性
	return DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Unscoped().Where("user_id = ?", t.UserId).Delete(&TwoFADevice{}).Error; err != nil {
			return err
		}

		// 同时删除相关的备用码记录（硬删除）
		if err := tx.Unscoped().Where("user_id = ?", t.UserId).Delete(&TwoFABackupCode{}).Error; err != nil {
			return err
		}

		// 硬删除2FA记录
		return tx.Unscoped().Delete(t).Error
	})
}

// ResetFailedAttempts 重置失败尝试次数
func (t *TwoFA) ResetFailedAttempts() error {
	t.FailedAttempts = 0
	t.LockedUntil = nil
	return t.Update()
}

// IncrementFailedAttempts 增加失败尝试次数
func (t *TwoFA) IncrementFailedAttempts() error {
	if t.Id == 0 {
		return errors.New("2FA记录ID不能为空")
	}

	const maxUpdateRetries = 5
	for range maxUpdateRetries {
		var current TwoFA
		if err := DB.Select("id", "failed_attempts", "locked_until").First(&current, t.Id).Error; err != nil {
			return err
		}

		now := time.Now()
		if current.LockedUntil != nil && now.Before(*current.LockedUntil) {
			t.FailedAttempts = current.FailedAttempts
			t.LockedUntil = current.LockedUntil
			return nil
		}

		nextFailedAttempts := current.FailedAttempts + 1
		nextLockedUntil := current.LockedUntil
		if nextFailedAttempts >= common.MaxFailAttempts {
			lockUntil := now.Add(time.Duration(common.LockoutDuration) * time.Second)
			nextLockedUntil = &lockUntil
		}

		result := DB.Model(&TwoFA{}).
			Where("id = ? AND failed_attempts = ? AND (locked_until IS NULL OR locked_until <= ?)", current.Id, current.FailedAttempts, now).
			Updates(map[string]interface{}{
				"failed_attempts": nextFailedAttempts,
				"locked_until":    nextLockedUntil,
			})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			continue
		}

		t.FailedAttempts = nextFailedAttempts
		t.LockedUntil = nextLockedUntil
		return nil
	}

	return errors.New("更新2FA失败次数冲突，请重试")
}

// IsLocked 检查账户是否被锁定
func (t *TwoFA) IsLocked() bool {
	if t.LockedUntil == nil {
		return false
	}
	return time.Now().Before(*t.LockedUntil)
}

// CreateBackupCodes 创建备用码
func CreateBackupCodes(userId int, codes []string) error {
	return DB.Transaction(func(tx *gorm.DB) error {
		// 先删除现有的备用码
		if err := tx.Where("user_id = ?", userId).Delete(&TwoFABackupCode{}).Error; err != nil {
			return err
		}

		// 创建新的备用码记录
		for _, code := range codes {
			hashedCode, err := common.HashBackupCode(code)
			if err != nil {
				return err
			}

			backupCode := TwoFABackupCode{
				UserId:   userId,
				CodeHash: hashedCode,
				IsUsed:   false,
			}

			if err := tx.Create(&backupCode).Error; err != nil {
				return err
			}
		}

		return nil
	})
}

// ValidateBackupCode 验证并使用备用码
func ValidateBackupCode(userId int, code string) (bool, error) {
	if !common.ValidateBackupCode(code) {
		return false, errors.New("验证码或备用码不正确")
	}

	normalizedCode := common.NormalizeBackupCode(code)

	// 查找未使用的备用码
	var backupCodes []TwoFABackupCode
	if err := DB.Where("user_id = ? AND is_used = false", userId).Find(&backupCodes).Error; err != nil {
		return false, err
	}

	// 验证备用码
	for _, bc := range backupCodes {
		if common.ValidatePasswordAndHash(normalizedCode, bc.CodeHash) {
			now := time.Now()
			result := DB.Model(&TwoFABackupCode{}).
				Where("id = ? AND is_used = ?", bc.Id, false).
				Updates(map[string]interface{}{
					"is_used": true,
					"used_at": now,
				})
			if result.Error != nil {
				return false, result.Error
			}
			return result.RowsAffected == 1, nil
		}
	}

	return false, nil
}

// GetUnusedBackupCodeCount 获取未使用的备用码数量
func GetUnusedBackupCodeCount(userId int) (int, error) {
	var count int64
	err := DB.Model(&TwoFABackupCode{}).Where("user_id = ? AND is_used = false", userId).Count(&count).Error
	return int(count), err
}

// DisableTwoFA 禁用用户的2FA
func DisableTwoFA(userId int) error {
	twoFA, err := GetTwoFAByUserId(userId)
	if err != nil {
		return err
	}
	if twoFA == nil {
		return ErrTwoFANotEnabled
	}

	// 删除2FA设置和备用码
	return twoFA.Delete()
}

// EnableTwoFA 启用2FA
func (t *TwoFA) Enable() error {
	t.IsEnabled = true
	t.FailedAttempts = 0
	t.LockedUntil = nil
	return t.Update()
}

func (t *TwoFA) markSuccessfulVerification(deviceId int) error {
	now := time.Now()
	t.FailedAttempts = 0
	t.LockedUntil = nil
	t.LastUsedAt = &now

	if err := t.Update(); err != nil {
		return err
	}

	if deviceId > LegacyPrimaryDeviceID {
		return DB.Model(&TwoFADevice{}).Where("id = ? AND user_id = ?", deviceId, t.UserId).
			Update("last_used_at", now).Error
	}
	return nil
}

// ValidateTOTPAndUpdateUsage 验证TOTP并更新使用记录（主验证器或任一额外设备均可）
func (t *TwoFA) ValidateTOTPAndUpdateUsage(code string) (bool, error) {
	if t.IsLocked() {
		return false, fmt.Errorf("账户已被锁定，请在%v后重试", t.LockedUntil.Format("2006-01-02 15:04:05"))
	}

	if t.IsEnabled && HasLegacyPrimarySecret(t) && common.ValidateTOTPCode(t.Secret, code) {
		if err := t.markSuccessfulVerification(LegacyPrimaryDeviceID); err != nil {
			common.SysLog("更新2FA使用记录失败: " + err.Error())
		}
		return true, nil
	}

	devices, err := GetEffectiveActiveTwoFADevices(t)
	if err != nil {
		return false, err
	}
	for _, device := range devices {
		if common.ValidateTOTPCode(device.Secret, code) {
			if err := t.markSuccessfulVerification(device.Id); err != nil {
				common.SysLog("更新2FA使用记录失败: " + err.Error())
			}
			return true, nil
		}
	}

	if err := t.IncrementFailedAttempts(); err != nil {
		common.SysLog("更新2FA失败次数失败: " + err.Error())
	}
	return false, nil
}

// ValidateLegacyTOTPAndUpdateUsage 验证主验证器 TOTP（用于首次启用）
func (t *TwoFA) ValidateLegacyTOTPAndUpdateUsage(code string) (bool, error) {
	if t.IsLocked() {
		return false, fmt.Errorf("账户已被锁定，请在%v后重试", t.LockedUntil.Format("2006-01-02 15:04:05"))
	}
	if !HasLegacyPrimarySecret(t) {
		return false, nil
	}
	if common.ValidateTOTPCode(t.Secret, code) {
		if err := t.markSuccessfulVerification(LegacyPrimaryDeviceID); err != nil {
			common.SysLog("更新2FA使用记录失败: " + err.Error())
		}
		return true, nil
	}
	if err := t.IncrementFailedAttempts(); err != nil {
		common.SysLog("更新2FA失败次数失败: " + err.Error())
	}
	return false, nil
}

// ValidateTOTPForDevice 验证指定额外设备的 TOTP 码（待确认或已启用）
func (t *TwoFA) ValidateTOTPForDevice(code string, deviceId int) (bool, error) {
	if deviceId <= LegacyPrimaryDeviceID {
		return false, errors.New("无效的设备ID")
	}
	if t.IsLocked() {
		return false, fmt.Errorf("账户已被锁定，请在%v后重试", t.LockedUntil.Format("2006-01-02 15:04:05"))
	}

	device, err := GetTwoFADeviceById(t.UserId, deviceId)
	if err != nil {
		return false, err
	}
	if device == nil {
		return false, nil
	}
	if common.ValidateTOTPCode(device.Secret, code) {
		if err := t.markSuccessfulVerification(device.Id); err != nil {
			common.SysLog("更新2FA使用记录失败: " + err.Error())
		}
		return true, nil
	}
	if err := t.IncrementFailedAttempts(); err != nil {
		common.SysLog("更新2FA失败次数失败: " + err.Error())
	}
	return false, nil
}

// ValidateBackupCodeAndUpdateUsage 验证备用码并更新使用记录
func (t *TwoFA) ValidateBackupCodeAndUpdateUsage(code string) (bool, error) {
	// 检查是否被锁定
	if t.IsLocked() {
		return false, fmt.Errorf("账户已被锁定，请在%v后重试", t.LockedUntil.Format("2006-01-02 15:04:05"))
	}

	// 验证备用码
	valid, err := ValidateBackupCode(t.UserId, code)
	if err != nil {
		return false, err
	}

	if !valid {
		// 增加失败次数
		if err := t.IncrementFailedAttempts(); err != nil {
			common.SysLog("更新2FA失败次数失败: " + err.Error())
		}
		return false, nil
	}

	if err := t.markSuccessfulVerification(0); err != nil {
		common.SysLog("更新2FA使用记录失败: " + err.Error())
	}

	return true, nil
}

// GetTwoFAStats 获取2FA统计信息（管理员使用）
func GetTwoFAStats() (map[string]interface{}, error) {
	var totalUsers, enabledUsers int64

	// 总用户数
	if err := DB.Model(&User{}).Count(&totalUsers).Error; err != nil {
		return nil, err
	}

	// 启用2FA的用户数
	if err := DB.Model(&TwoFA{}).Where("is_enabled = true").Count(&enabledUsers).Error; err != nil {
		return nil, err
	}

	enabledRate := float64(0)
	if totalUsers > 0 {
		enabledRate = float64(enabledUsers) / float64(totalUsers) * 100
	}

	return map[string]interface{}{
		"total_users":   totalUsers,
		"enabled_users": enabledUsers,
		"enabled_rate":  fmt.Sprintf("%.1f%%", enabledRate),
	}, nil
}
