package repository

import (
	"context"
	"errors"
	"log/slog"
	"strconv"

	userDomain "github.com/UniPro-tech/UniQUE-API/api/internal/domain/user"
	"github.com/UniPro-tech/UniQUE-API/api/internal/driver/mysql/scheme"
	"github.com/UniPro-tech/UniQUE-API/api/pkg"

	"gorm.io/gorm"
)

type UserDriver struct {
	conn *gorm.DB
}

// Delete implements user.UserServiceRepository.
func (ud *UserDriver) Delete(ctx context.Context, id string) error {
	ctxValue := ctx.Value("ctxInfo").(pkg.CtxInfo)
	user := scheme.User{}
	err := ud.conn.Where("uid = ?", id).Delete(&user).Error
	if err != nil {
		slog.Info("can not complete DeleteUser Repository", "request id", ctxValue.RequestId)
		return err
	}

	slog.Info("process done DeleteUser Repository", "request id", ctxValue.RequestId)
	return nil
}

// ListUser implements user.UserServiceRepository.
func (ud *UserDriver) ListUser(ctx context.Context) ([]*userDomain.User, int64, error) {
	ctxValue := ctx.Value("ctxInfo").(pkg.CtxInfo)
	res := []*userDomain.User{}
	users := []*scheme.User{}
	var totalCount int64

	limit, err := strconv.Atoi(ctxValue.PageLimit)
	if err != nil {
		limit = 100
	}
	page, err := strconv.Atoi(ctxValue.Pages)
	if err != nil {
		page = 0
	}

	err = ud.conn.Table("users").Limit(limit).Offset(limit * (page - 1)).Order("id ASC").Find(&users).Count(&totalCount).Error
	if err != nil {
		slog.Error("can not complate FindByID Repository", "error msg", err, "request id", ctxValue.RequestId)
		return nil, 0, err
	}

	for _, user := range users {
		u := userDomain.NewUser(user.ID, user.Email, user.CustomID, user.Name, user.ExternalEmail, user.Period, &user.IsEnable)

		res = append(res, u)
	}

	slog.Info("process done ListUser Repository", "request id", ctxValue.RequestId, "total count", totalCount)
	return res, totalCount, nil
}

// FindUserById implements user.UserServiceRepository.
func (ud *UserDriver) FindUserById(ctx context.Context, id string) (*userDomain.User, error) {
	ctxValue := ctx.Value("ctxInfo").(pkg.CtxInfo)
	user := scheme.User{}

	err := ud.conn.Table("users").Where("id = ?", id).Find(&user).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		slog.Error("can not complate FindByID Repository", "request id", ctxValue.RequestId)
		return nil, gorm.ErrRecordNotFound
	}

	res := userDomain.NewUser(user.ID, user.Email, user.CustomID, user.Name, user.ExternalEmail, user.Period, &user.IsEnable)
	if err != nil {
		slog.Error("can not complete FindByID Repository", "request id", ctxValue.RequestId, "error", err)
		return nil, err
	}

	slog.Info("process done FindByID Repository", "request id", ctxValue.RequestId, "user", user)
	return res, nil
}

// Save implements user.UserServiceRepository.
func (ud *UserDriver) Save(ctx context.Context, param *userDomain.User) error {
	ctxValue := ctx.Value("ctxInfo").(pkg.CtxInfo)

	repoUser := scheme.User{
		ID:       param.GetID(),
		CustomID: param.GetCustomID(),
		Name:     param.GetName(),
		Period:   param.GetPeriod(),
		IsEnable: param.GetIsEnable(),
		Email:    param.GetEmail(),
	}

	err := ud.conn.Table("users").Save(&repoUser).Error
	if err != nil {
		slog.Error("can not complete SaveUser Repository", "request id", ctxValue.RequestId)
		return err
	}

	slog.Info("process done SaveUser Repository", "request id", ctxValue.RequestId)
	return nil
}

func NewUserDriver(conn *gorm.DB) userDomain.UserServiceRepository {
	return &UserDriver{conn: conn}
}
