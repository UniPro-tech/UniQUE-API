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

// Deleteは、指定されたユーザーIDに基づいてデータベースからユーザーのレコードを削除します。
// コンテキストから取得したリクエストIDを使用して、操作状況をログに記録します。
// 削除に失敗した場合はエラーを返します。
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

// ListUserは、データベースからユーザーのページネーションされた一覧を取得します。
// コンテキストからページ制限やページ番号などのページネーション情報を利用します。
// userDomain.Userのスライス、ユーザーの総件数、およびクエリ中に発生したエラーを返します。
// ページネーションのパラメータが不正な場合は、デフォルト値が使用されます。
// エラーや処理状況はslogでログに記録されます。
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

// FindUserByIdは、指定されたユーザーIDに基づいてデータベースからユーザー情報を取得します。
// コンテキストとユーザーIDを受け取り、Userドメインオブジェクトのポインタとエラーを返します。
// ユーザーが見つからない場合はgorm.ErrRecordNotFoundを返します。
// エラーや処理状況は、コンテキストから取得したリクエストIDとともにslogでログに記録します。
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

// Save は、指定されたユーザードメインオブジェクトの情報をデータベースに上書き保存します。
// コンテキスト情報（ctxInfo）を利用してリクエストIDを取得し、処理のログを記録します。
// 保存処理に失敗した場合はエラーを返します。
//
// param:
//
//	ctx   - リクエストのコンテキスト情報
//	param - 保存対象のユーザードメインオブジェクト
//
// returns:
//
//	error - 保存処理に失敗した場合のエラー
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

// Create は、新しいユーザー情報をデータベースに登録します。
//
// 引数:
//
//	ctx   - コンテキスト情報を保持する context.Context 型。
//	param - 登録するユーザー情報を保持する userDomain.User 型のポインタ。
//
// 戻り値:
//
//	error - 登録処理に失敗した場合はエラーを返します。
//	        主な失敗要因として、重複エントリ（MySQL エラーコード 1062）や
//	        データベースエラーが考えられます。
//
// 備考:
//
//	登録処理の成否に応じて、適切なログ出力を行います。
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

// Update は、指定されたユーザ情報（userDomain.User）をデータベース上の該当ユーザのレコードの特定のカラムのみ更新します。
// ctx にはリクエストコンテキスト情報が含まれており、更新処理のログ出力に利用されます。
// param には更新対象のユーザ情報が格納されています。
// 更新処理が正常に完了した場合は nil を返し、エラーが発生した場合はそのエラーを返します。
// エラー発生時にはエラーログを、正常終了時には完了ログを出力します。
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

// Search は、指定された検索パラメータに基づいてユーザー情報を検索し、
// 該当するユーザーのリストと総件数を返します。
//
// 引数:
//
//	ctx - コンテキスト情報（リクエストIDやページ情報などを含む）
//	searchParams - ユーザー検索条件を格納した構造体
//
// 戻り値:
//
//	[]*userDomain.User - 検索結果のユーザー情報のスライス
//	int64 - 検索結果の総件数
//	error - 検索処理中に発生したエラー（存在
//
// 検索条件にはID、メールアドレス、カスタムID、氏名、外部メールアドレス、期間、
// 有効/無効フラグなどが指定可能です。
// ページネーションとソート（ID昇順）にも対応しています。
// 検索処理中にエラーが発生した場合は、エラー情報を返します。
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
