package user

import (
	"context"
	"errors"
	"regexp"
	"time"

	"github.com/UniPro-tech/UniQUE-API/api/pkg"
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
type User struct {
	id             userUUID
	email          userInternalEmail
	custom_id      customID
	name           userName
	external_email userExternalEmail
	period         userPeriod
	is_enable      userIsEnable
	password_hash  userPasswordHash
	created_at     *userCreatedAt
	updated_at     *userUpdatedAt
	joined_at      *userJoinedAt
}

// ドメイン バリューオブジェクト
type userUUID struct{ value string }
type customID struct{ value string }
type userInternalEmail struct{ value string }
type userExternalEmail struct{ value string }
type userName struct{ value string }
type userPeriod struct{ value string }
type userIsEnable struct{ value bool }
type userPasswordHash struct{ value string }
type userCreatedAt struct{ value time.Time }
type userUpdatedAt struct{ value time.Time }
type userJoinedAt struct{ value time.Time }

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

/* userExternalEmail バリデーション godoc メールアドレスの形式のなっていること */
func (v *userExternalEmail) Valid() error {
	match, _ := regexp.MatchString(USER_EXTERNAL_EMAIL_PATTERN, v.value)
	if !match {
		return ERR_INVALID_EXTERNAL_EMAIL
	}

	return nil
}

/* userInternalEmail バリデーション godoc
* メールアドレスが "period.custom_id@uniproject.jp" の形式であること
 */
func (v *userInternalEmail) Valid(customID string, period string) error {
	expectedEmail := period + "." + customID + "@uniproject.jp"
	if period == "0" {
		expectedEmail = customID + "@uniproject.jp"
	}
	if v.value != expectedEmail {
		return errors.New("invalid internal email address")
	}
	return nil
}

func (v *User) Valid() error {
	if err := v.custom_id.Valid(); err != nil {
		return ERR_INVALID_CUSTOM_ID
	}
	if err := v.email.Valid(v.GetCustomID(), v.GetPeriod()); err != nil {
		return ERR_INVALID_EMAIL
	}
	if err := v.external_email.Valid(); err != nil {
		return ERR_INVALID_EXTERNAL_EMAIL
	}
	return nil
}

// バリューオブジェクトの取得関数
func (u *User) GetID() string            { return u.id.value }
func (u *User) GetCustomID() string      { return u.custom_id.value }
func (u *User) GetEmail() string         { return u.email.value }
func (u *User) GetName() string          { return u.name.value }
func (u *User) GetExternalEmail() string { return u.external_email.value }
func (u *User) GetPeriod() string        { return u.period.value }
func (u *User) GetIsEnable() bool        { return u.is_enable.value }
func (u *User) GetPasswordHash() string {
	if u.password_hash.value == "" {
		return ""
	}
	return u.password_hash.value
}
func (u *User) GetCreatedAt() time.Time {
	if u.created_at == nil {
		return time.Time{}
	}
	return u.created_at.value
}
func (u *User) GetUpdatedAt() time.Time {
	if u.updated_at == nil {
		return time.Time{}
	}
	return u.updated_at.value
}
func (u *User) GetJoinedAt() time.Time {
	if u.joined_at == nil {
		return time.Time{}
	}
	return u.joined_at.value
}

// 構造体生成関数
func NewUser(id string, name string, email string, custom_id string, externalEmail string, period string, is_enable bool, password_hash *string, joined_at *time.Time) *User {
	if password_hash == nil {
		password_hash = new(string)
	}
	return newUser(id, email, custom_id, name, externalEmail, period, is_enable, password_hash, joined_at)
}

func newUser(id string, email string, custom_id string, name string, externalEmail string, period string, is_enable bool, password_hash *string, joined_at *time.Time) *User {
	if password_hash == nil || *password_hash == "" {
		password_hash = new(string)
	}
	if joined_at == nil {
		return &User{
			id:             userUUID{value: id},
			email:          userInternalEmail{value: email},
			custom_id:      customID{value: custom_id},
			name:           userName{value: name},
			external_email: userExternalEmail{value: externalEmail},
			period:         userPeriod{value: period},
			is_enable:      userIsEnable{value: is_enable},
			password_hash:  userPasswordHash{value: *password_hash},
		}
	}

	return &User{
		id:             userUUID{value: id},
		email:          userInternalEmail{value: email},
		custom_id:      customID{value: custom_id},
		name:           userName{value: name},
		external_email: userExternalEmail{value: externalEmail},
		period:         userPeriod{value: period},
		is_enable:      userIsEnable{value: is_enable},
		password_hash:  userPasswordHash{value: *password_hash},
		joined_at:      &userJoinedAt{value: *joined_at},
	}
}

//go:generate moq -out IUserDomainService_mock.go . IUserDomainService
type IUserDomainService interface {
	ListUser(ctx context.Context) ([]*User, int64, error)
	FindUserById(ctx context.Context, id string) (*User, error)
	SearchUser(ctx context.Context, searchParams pkg.UserParams) ([]*User, int64, error)
	EditUser(ctx context.Context, param *User) error
	DeleteUser(ctx context.Context, id string) error
	AddUser(ctx context.Context, param *User) error
	SaveUser(ctx context.Context, param *User) error
	UpdateUser(ctx context.Context, param *User) error
}
