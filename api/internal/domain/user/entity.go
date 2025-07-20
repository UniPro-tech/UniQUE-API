package user

import (
	"context"
	"errors"
	"regexp"

	"github.com/UniPro-tech/UniQUE-API/api/pkg"
)

const (
	CUSTOM_ID_PATTERN           = `^[a-zA-Z0-9](?:(?<![-_])[a-zA-Z0-9]|[-_](?![-_])){0,9}$`
	USER_EXTERNAL_EMAIL_PATTERN = `^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`
)

var (
	ErrInvalidCustomID  = errors.New("invalid custom id")
	ErrUserEmailAddress = errors.New("invalid user email address")
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
}

// ドメイン バリューオブジェクト
type userUUID struct{ value string }
type customID struct{ value string }
type userInternalEmail struct{ value string }
type userExternalEmail struct{ value string }
type userName struct{ value string }
type userPeriod struct{ value string }
type userIsEnable struct{ value bool }

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

	// 結果を出力
	if !matched {
		return ErrInvalidCustomID
	}

	return nil
}

/* userExternalEmail バリデーション godoc メールアドレスの形式のなっていること */
func (v *userExternalEmail) Valid() error {
	match, _ := regexp.MatchString(USER_EXTERNAL_EMAIL_PATTERN, v.value)
	if !match {
		return ErrUserEmailAddress
	}

	return nil
}

/* userInternalEmail バリデーション godoc
* メールアドレスが "period.custom_id@uniproject.jp" の形式であること
 */
func (v *userInternalEmail) Valid(customID string, period string) error {
	expectedEmail := period + "." + customID + "@uniproject.jp"
	if v.value != expectedEmail {
		return errors.New("invalid internal email address")
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

// 構造体生成関数
func NewUser(id string, name string, email string, custom_id string, externalEmail string, period string, is_enable *bool) *User {
	return newUser(id, email, custom_id, name, externalEmail, period, is_enable)
}

func newUser(id string, email string, custom_id string, name string, externalEmail string, period string, is_enable *bool) *User {
	if is_enable == nil {
		defaultEnable := true
		is_enable = &defaultEnable
	}
	return &User{
		id:             userUUID{value: id},
		email:          userInternalEmail{value: email},
		custom_id:      customID{value: custom_id},
		name:           userName{value: name},
		external_email: userExternalEmail{value: externalEmail},
		period:         userPeriod{value: period},
		is_enable:      userIsEnable{value: *is_enable},
	}
}

//go:generate moq -out IUserDomainService_mock.go . IUserDomainService
type IUserDomainService interface {
	ListUser(ctx context.Context) ([]*User, int64, error)
	FindUserById(ctx context.Context, id string) (*User, error)
	SearchUser(ctx context.Context, searchParams pkg.UserSearchParams) ([]*User, int64, error)
	EditUser(ctx context.Context, param *User) error
	DeleteUser(ctx context.Context, id string) error
}
