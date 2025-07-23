package repository

import (
	"context"
	"errors"
	"log/slog"
	"strconv"

	userDomain "github.com/UniPro-tech/UniQUE-API/api/internal/domain/user"
	sqlerrors "github.com/UniPro-tech/UniQUE-API/api/internal/driver/mysql/errors"
	"github.com/UniPro-tech/UniQUE-API/api/internal/driver/mysql/scheme"
	"github.com/UniPro-tech/UniQUE-API/api/pkg"
	"github.com/go-sql-driver/mysql"

	"gorm.io/gorm"
)

type UserDriver struct {
	conn *gorm.DB
}

// Delete implements user.UserServiceRepository.
func (ud *UserDriver) Delete(ctx context.Context, id string) error {
	ctxValue := ctx.Value("ctxInfo").(pkg.CtxInfo)
	user := scheme.User{}
	err := ud.conn.Where("id = ?", id).Delete(&user).Error
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
		u := userDomain.NewUser(user.ID, user.Name, user.Email, user.CustomID, user.ExternalEmail, user.Period, user.IsEnable)

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

	res := userDomain.NewUser(user.ID, user.Email, user.CustomID, user.Name, user.ExternalEmail, user.Period, user.IsEnable)
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

	repoUser := &scheme.User{
		ID:            param.GetID(),
		CustomID:      param.GetCustomID(),
		Name:          param.GetName(),
		Period:        param.GetPeriod(),
		IsEnable:      param.GetIsEnable(),
		ExternalEmail: param.GetExternalEmail(),
		Email:         param.GetEmail(),
		PasswordHash:  param.GetPasswordHash(),
	}

	err := ud.conn.Table("users").Save(repoUser).Error
	if err != nil {
		slog.Error("can not complete SaveUser Repository", "request id", ctxValue.RequestId)
		return err
	}

	slog.Info("process done SaveUser Repository", "request id", ctxValue.RequestId)
	return nil
}

func (ud *UserDriver) Create(ctx context.Context, param *userDomain.User) error {
	ctxValue := ctx.Value("ctxInfo").(pkg.CtxInfo)

	repoUser := &scheme.User{
		ID:            param.GetID(),
		CustomID:      param.GetCustomID(),
		Name:          param.GetName(),
		Period:        param.GetPeriod(),
		IsEnable:      param.GetIsEnable(),
		ExternalEmail: param.GetExternalEmail(),
		Email:         param.GetEmail(),
		PasswordHash:  param.GetPasswordHash(),
	}

	err := ud.conn.Table("users").Create(repoUser).Error
	if err != nil {
		if mysqlErr, ok := err.(*mysql.MySQLError); ok {
			switch mysqlErr.Number {
			case 1062:
				slog.Error("Duplicate entry error in CreateUser Repository", "request id", ctxValue.RequestId, "error", mysqlErr)
				return sqlerrors.ERR_DUPLICATE_ENTRY
			default:
				slog.Error("can not complete CreateUser Repository", "request id", ctxValue.RequestId, "error", err)
				return errors.New("failed to create user due to database error")
			}
		} else {
			slog.Error("can not complete CreateUser Repository", "request id", ctxValue.RequestId, "error", err)
			return errors.New("failed to create user due to database error")
		}
	}

	slog.Info("process done CreateUser Repository", "request id", ctxValue.RequestId)
	return nil
}

func (ud *UserDriver) Update(ctx context.Context, param *userDomain.User) error {
	ctxValue := ctx.Value("ctxInfo").(pkg.CtxInfo)

	repoUser := &scheme.User{
		ID:            param.GetID(),
		CustomID:      param.GetCustomID(),
		Name:          param.GetName(),
		Period:        param.GetPeriod(),
		IsEnable:      param.GetIsEnable(),
		ExternalEmail: param.GetExternalEmail(),
		Email:         param.GetEmail(),
		PasswordHash:  param.GetPasswordHash(),
	}

	err := ud.conn.Table("users").Updates(repoUser).Error
	if err != nil {
		slog.Error("can not complete UpdateUser Repository", "request id", ctxValue.RequestId)
		return err
	}

	slog.Info("process done UpdateUser Repository", "request id", ctxValue.RequestId)
	return nil
}

func (ud *UserDriver) Search(ctx context.Context, searchParams pkg.UserParams) ([]*userDomain.User, int64, error) {
	ctxValue := ctx.Value("ctxInfo").(pkg.CtxInfo)
	users := []*scheme.User{}
	res := []*userDomain.User{}
	var totalCount int64

	query := &scheme.User{
		ID:            *searchParams.ID,
		Email:         *searchParams.Email,
		CustomID:      *searchParams.CustomID,
		Name:          *searchParams.Name,
		ExternalEmail: *searchParams.ExternalEmail,
		Period:        *searchParams.Period,
	}
	if searchParams.IsEnable != nil {
		if *searchParams.IsEnable == "true" {
			query.IsEnable = true
		} else {
			query.IsEnable = false
		}
	}

	limit, err := strconv.Atoi(ctxValue.PageLimit)
	if err != nil {
		limit = 100
	}
	page, err := strconv.Atoi(ctxValue.Pages)
	if err != nil {
		page = 0
	}

	err = ud.conn.Table("users").Where(query).Limit(limit).Offset(limit * (page - 1)).Order("id ASC").Find(&users).Count(&totalCount).Error
	if err != nil {
		slog.Error("can not complete SearchUser Repository", "request id", ctxValue.RequestId, "error", err)
		return nil, 0, err
	}

	for _, user := range users {
		u := userDomain.NewUser(user.ID, user.Name, user.Email, user.CustomID, user.ExternalEmail, user.Period, user.IsEnable)

		res = append(res, u)
	}

	slog.Info("process done ListUser Repository", "request id", ctxValue.RequestId, "total count", totalCount, "users", res)
	return res, totalCount, nil
}

func NewUserDriver(conn *gorm.DB) userDomain.UserServiceRepository {
	return &UserDriver{conn: conn}
}
