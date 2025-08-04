package role

import (
	"errors"
	"regexp"
	"time"
)

const (
	CUSTOM_ID_PATTERN           = `^[a-zA-Z0-9-_]{1,10}$`
	USER_EXTERNAL_EMAIL_PATTERN = `^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`
)

var (
	ERR_INVALID_CUSTOM_ID      = errors.New("invalid custom id")
	ERR_INVALID_EMAIL          = errors.New("invalid user email address")
	ERR_INVALID_EXTERNAL_EMAIL = errors.New("invalid external email address")
)

// ドメインモデル
type Role struct {
	id         roleUUID
	name       roleName
	custom_id  customID
	permission rolePermission
	created_at roleCreatedAt
	updated_at *roleUpdatedAt
	is_enable  roleIsEnable
	is_system  roleIsSystem
}

// ドメイン バリューオブジェクト
type roleUUID struct{ value string }
type customID struct{ value string }
type roleName struct{ value string }
type roleIsEnable struct{ value bool }
type roleCreatedAt struct{ value time.Time }
type roleUpdatedAt struct{ value time.Time }
type roleIsSystem struct{ value bool }
type rolePermission struct {
	value []string
	bits  uint32
}

// ドメインルール

/*
userID バリデーション godoc
* 10文字
* 英数字
* 記号なし
*/
func (v *customID) Valid() error {
	r := regexp.MustCompile(CUSTOM_ID_PATTERN)
	matched := r.MatchString(v.value)
	if !matched {
		return ERR_INVALID_CUSTOM_ID
	}
	if v.value[0] == '-' || v.value[0] == '_' {
		return ERR_INVALID_CUSTOM_ID
	}
	if v.value[len(v.value)-1] == '-' || v.value[len(v.value)-1] == '_' {
		return ERR_INVALID_CUSTOM_ID
	}
	for i := 1; i < len(v.value); i++ {
		if (v.value[i] == '-' || v.value[i] == '_') && (v.value[i-1] == '-' || v.value[i-1] == '_') {
			return ERR_INVALID_CUSTOM_ID
		}
	}
	return nil
}

func (v *roleName) Valid() error {
	if len(v.value) == 0 || len(v.value) > 50 {
		return errors.New("role name must be between 1 and 50 characters")
	}
	for _, r := range v.value {
		// 制御文字や記号（絵文字含む）を弾く
		if r < 0x20 || (r >= 0x7F && r <= 0xA0) || (r >= 0x2000 && r <= 0x2FFF) {
			return errors.New("role name contains invalid characters")
		}
	}
	return nil
}

func (v *Role) Valid() error {

	return nil
}

// バリューオブジェクトの取得関数
func (r *Role) GetID() string       { return r.id.value }
func (r *Role) GetCustomID() string { return r.custom_id.value }
func (r *Role) GetName() string     { return r.name.value }
func (r *Role) GetIsEnable() bool   { return r.is_enable.value }
func (r *Role) GetCreatedAt() time.Time {
	return r.created_at.value
}
func (r *Role) GetUpdatedAt() time.Time {
	if r.updated_at == nil {
		return time.Time{}
	}
	return r.updated_at.value
}
func (r *Role) GetIsSystem() bool            { return r.is_system.value }
func (r *Role) GetPermissionArray() []string { return r.permission.value }
func (r *Role) GetPermissionBits() uint32    { return r.permission.bits }

// 構造体生成関数
func NewRole(id string, custom_id string, name string, is_enable bool, is_system bool, permission []string) *Role {
	return newRole(id, custom_id, name, is_enable, is_system, permission, 0)
}

func newRole(id string, custom_id string, name string, is_enable bool, is_system bool, permission []string, permissionBit uint32) *Role {
	return &Role{
		id:         roleUUID{value: id},
		custom_id:  customID{value: custom_id},
		name:       roleName{value: name},
		permission: rolePermission{value: permission, bits: 0},
		is_enable:  roleIsEnable{value: is_enable},
		is_system:  roleIsSystem{value: is_system},
	}
}

//go:generate moq -out IUserDomainService_mock.go . IUserDomainService
type IRoleDomainService interface {
}
