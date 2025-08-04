package repository

import (
	"context"
	"errors"
	"log/slog"
	"strconv"

	roleDomain "github.com/UniPro-tech/UniQUE-API/api/internal/domain/role"
	sqlerrors "github.com/UniPro-tech/UniQUE-API/api/internal/driver/mysql/errors"
	"github.com/UniPro-tech/UniQUE-API/api/internal/driver/mysql/scheme"
	"github.com/UniPro-tech/UniQUE-API/api/pkg"
	"github.com/go-sql-driver/mysql"

	"gorm.io/gorm"
)

type RoleDriver struct {
	conn *gorm.DB
}

// Deleteは、指定されたユーザーIDに基づいてデータベースからユーザーのレコードを削除します。
// コンテキストから取得したリクエストIDを使用して、操作状況をログに記録します。
// 削除に失敗した場合はエラーを返します。
func (rd *RoleDriver) Delete(ctx context.Context, id string) error {
	ctxValue := ctx.Value("ctxInfo").(pkg.CtxInfo)
	user := scheme.User{}
	err := rd.conn.Where("id = ?", id).Delete(&user).Error
	if err != nil {
		slog.Info("can not complete DeleteUser Repository", "request id", ctxValue.RequestId)
		return err
	}

	slog.Info("process done DeleteUser Repository", "request id", ctxValue.RequestId)
	return nil
}

// ListUserは、データベースからユーザーのページネーションされた一覧を取得します。
// コンテキストからページ制限やページ番号などのページネーション情報を利用します。
// roleDomain.Roleのスライス、ユーザーの総件数、およびクエリ中に発生したエラーを返します。
// ページネーションのパラメータが不正な場合は、デフォルト値が使用されます。
// エラーや処理状況はslogでログに記録されます。
func (rd *RoleDriver) ListRole(ctx context.Context) ([]*roleDomain.Role, int64, error) {
	ctxValue := ctx.Value("ctxInfo").(pkg.CtxInfo)
	res := []*roleDomain.Role{}
	roles := []*scheme.Role{}
	var totalCount int64

	limit, err := strconv.Atoi(ctxValue.PageLimit)
	if err != nil {
		limit = 100
	}
	page, err := strconv.Atoi(ctxValue.Pages)
	if err != nil {
		page = 0
	}

	err = rd.conn.Table("roles").Limit(limit).Offset(limit * (page - 1)).Order("id ASC").Find(&roles).Count(&totalCount).Error
	if err != nil {
		slog.Error("can not complate FindByID Repository", "error msg", err, "request id", ctxValue.RequestId)
		return nil, 0, err
	}

	for _, role := range roles {
		// TODO: 実装　role.Permission
		u := roleDomain.NewRole(role.ID, role.CustomID, role.Name, role.IsEnable, role.IsSystem, []string{})

		res = append(res, u)
	}

	slog.Info("process done ListUser Repository", "request id", ctxValue.RequestId, "total count", totalCount)
	return res, totalCount, nil
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
func (rd *RoleDriver) Save(ctx context.Context, param *roleDomain.Role) error {
	ctxValue := ctx.Value("ctxInfo").(pkg.CtxInfo)

	repoRole := &scheme.Role{
		ID:         param.GetID(),
		CustomID:   param.GetCustomID(),
		Name:       param.GetName(),
		IsEnable:   param.GetIsEnable(),
		Permission: int32(param.GetPermissionBits()),
	}

	err := rd.conn.Table("roles").Save(repoRole).Error
	if err != nil {
		slog.Error("can not complete SaveRole Repository", "request id", ctxValue.RequestId)
		return err
	}

	slog.Info("process done SaveRole Repository", "request id", ctxValue.RequestId)
	return nil
}

// Create は、新しいユーザー情報をデータベースに登録します。
//
// 引数:
//
//	ctx   - コンテキスト情報を保持する context.Context 型。
//	param - 登録するユーザー情報を保持する roleDomain.Role 型のポインタ。
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
func (rd *RoleDriver) Create(ctx context.Context, param *roleDomain.Role) error {
	ctxValue := ctx.Value("ctxInfo").(pkg.CtxInfo)

	repoRole := &scheme.Role{
		ID:         param.GetID(),
		CustomID:   param.GetCustomID(),
		Name:       param.GetName(),
		IsEnable:   param.GetIsEnable(),
		Permission: int32(param.GetPermissionBits()),
		IsSystem:   param.GetIsSystem(),
	}

	err := rd.conn.Table("roles").Create(repoRole).Error
	if err != nil {
		if mysqlErr, ok := err.(*mysql.MySQLError); ok {
			switch mysqlErr.Number {
			case 1062:
				slog.Error("Duplicate entry error in CreateRole Repository", "request id", ctxValue.RequestId, "error", mysqlErr)
				return sqlerrors.ERR_DUPLICATE_ENTRY
			default:
				slog.Error("can not complete CreateRole Repository", "request id", ctxValue.RequestId, "error", err)
				return errors.New("failed to create role due to database error")
			}
		}
	}
	err = rd.conn.Table("roles").Create(repoRole).Error
	if err != nil {
		if mysqlErr, ok := err.(*mysql.MySQLError); ok {
			switch mysqlErr.Number {
			case 1062:
				slog.Error("Duplicate entry error in CreateRole Repository", "request id", ctxValue.RequestId, "error", mysqlErr)
				return sqlerrors.ERR_DUPLICATE_ENTRY
			default:
				slog.Error("can not complete CreateRole Repository", "request id", ctxValue.RequestId, "error", err)
				return errors.New("failed to create role due to database error")
			}
		} else {
			slog.Error("can not complete CreateUser Repository", "request id", ctxValue.RequestId, "error", err)
			return errors.New("failed to create user due to database error")
		}
	}

	slog.Info("process done CreateUser Repository", "request id", ctxValue.RequestId)
	return nil
}

// Update は、指定されたユーザ情報（roleDomain.Role）をデータベース上の該当ユーザのレコードの特定のカラムのみ更新します。
// ctx にはリクエストコンテキスト情報が含まれており、更新処理のログ出力に利用されます。
// param には更新対象のユーザ情報が格納されています。
// 更新処理が正常に完了した場合は nil を返し、エラーが発生した場合はそのエラーを返します。
// エラー発生時にはエラーログを、正常終了時には完了ログを出力します。
func (rd *RoleDriver) Update(ctx context.Context, param *roleDomain.Role) error {
	ctxValue := ctx.Value("ctxInfo").(pkg.CtxInfo)

	repoRole := &scheme.Role{
		ID:         param.GetID(),
		CustomID:   param.GetCustomID(),
		Name:       param.GetName(),
		IsEnable:   param.GetIsEnable(),
		Permission: int32(param.GetPermissionBits()),
		IsSystem:   param.GetIsSystem(),
	}

	err := rd.conn.Table("roles").Updates(repoRole).Error
	if err != nil {
		slog.Error("can not complete UpdateRole Repository", "request id", ctxValue.RequestId)
		return err
	}

	slog.Info("process done UpdateRole Repository", "request id", ctxValue.RequestId)
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
//	[]*roleDomain.Role - 検索結果のユーザー情報のスライス
//	int64 - 検索結果の総件数
//	error - 検索処理中に発生したエラー（存在
//
// 検索条件にはID、メールアドレス、カスタムID、氏名、外部メールアドレス、期間、
// 有効/無効フラグなどが指定可能です。
// ページネーションとソート（ID昇順）にも対応しています。
// 検索処理中にエラーが発生した場合は、エラー情報を返します。
func (rd *RoleDriver) Search(ctx context.Context, searchParams pkg.RoleParams) ([]*roleDomain.Role, int64, error) {
	ctxValue := ctx.Value("ctxInfo").(pkg.CtxInfo)
	roles := []*scheme.Role{}
	res := []*roleDomain.Role{}
	var totalCount int64

	query := &scheme.Role{
		ID:       *searchParams.ID,
		CustomID: *searchParams.CustomID,
		Name:     *searchParams.Name,
	}
	if searchParams.IsEnable != nil {
		if *searchParams.IsEnable == "true" {
			query.IsEnable = true
		} else {
			query.IsEnable = false
		}
	}
	if searchParams.IsSystem != nil {
		if *searchParams.IsSystem == "true" {
			query.IsSystem = true
		} else {
			query.IsSystem = false
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

	err = rd.conn.Table("roles").Where(query).Limit(limit).Offset(limit * (page - 1)).Order("id ASC").Find(&roles).Count(&totalCount).Error
	if err != nil {
		slog.Error("can not complete SearchRole Repository", "request id", ctxValue.RequestId, "error", err)
		return nil, 0, err
	}

	for _, role := range roles {
		// TODO: 実装する role.Permission
		r := roleDomain.NewRole(role.ID, role.Name, role.CustomID, role.IsEnable, role.IsSystem, []string{})

		res = append(res, r)
	}

	slog.Info("process done ListRole Repository", "request id", ctxValue.RequestId, "total count", totalCount, "roles", res)
	return res, totalCount, nil
}

func NewRoleDriver(conn *gorm.DB) roleDomain.RoleServiceRepository {
	return &RoleDriver{conn: conn}
}
