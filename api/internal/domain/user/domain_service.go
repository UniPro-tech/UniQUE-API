package user

import (
	"context"
	"log/slog"

	"github.com/UniPro-tech/UniQUE-API/api/pkg"
)

type UserDomainService struct {
	repo UserServiceRepository
}

func NewUserDomainService(repo UserServiceRepository) *UserDomainService {
	return &UserDomainService{repo: repo}
}

// ListUser は、ユーザーの一覧を取得するドメインサービスメソッドです。
// コンテキストを受け取り、ユーザー情報のスライス、ユーザー総数、およびエラーを返します。
// ユーザー情報の取得に失敗した場合は、nil と 0、およびエラーを返します。
func (uds *UserDomainService) ListUser(ctx context.Context) ([]*User, int64, error) {
	users, count, err := uds.repo.ListUser(ctx)
	if err != nil {
		return nil, 0, err
	}
	return users, count, nil
}

// FindUserById は、指定されたユーザーIDに基づいてユーザー情報を取得します。
// ctx はリクエストのコンテキストを表し、id は検索対象のユーザーIDです。
// ユーザーが見つかった場合は User オブジェクトを返し、見つからない場合やエラーが発生した場合は error を返します。
func (uds *UserDomainService) FindUserById(ctx context.Context, id string) (*User, error) {
	user, err := uds.repo.FindUserById(ctx, id)
	if err != nil {
		return nil, err
	}
	return user, nil
}

// EditUser は、指定されたユーザー情報を検証し、リポジトリに保存します。
// ユーザー情報が不正な場合はエラーを返します。
// 保存処理に失敗した場合もエラーを返します。
//
// 引数:
//
//	ctx - 操作のコンテキスト
//	param - 編集対象のユーザー情報
//
// 戻り値:
//
//	error - 保存処理の結果、エラーが発生した場合はそのエラーを返します。
func (uds *UserDomainService) EditUser(ctx context.Context, param *User) error {
	if err := param.Valid(); err != nil {
		return err
	}
	err := uds.repo.Save(ctx, param)
	if err != nil {
		return err
	}
	return nil
}

// DeleteUser は指定されたユーザーIDに対応するユーザーを削除します。
// ctx はリクエストのコンテキストを表し、id は削除対象ユーザーのIDです。
// 削除処理に失敗した場合はエラーを返します。
func (uds *UserDomainService) DeleteUser(ctx context.Context, id string) error {
	err := uds.repo.Delete(ctx, id)
	if err != nil {
		return err
	}
	return nil
}

// SearchUser は、指定された検索パラメータに基づいてユーザー情報を検索します。
// 検索結果としてユーザーのスライス、総件数、およびエラー情報を返します。
// ctx: コンテキスト情報。
// searchParams: ユーザー検索のためのパラメータ。
// 戻り値: ユーザー情報のスライス、総件数、エラー情報。
func (uds *UserDomainService) SearchUser(ctx context.Context, searchParams pkg.UserParams) ([]*User, int64, error) {
	users, count, err := uds.repo.Search(ctx, searchParams)
	if err != nil {
		return nil, 0, err
	}
	return users, count, nil
}

func (uds *UserDomainService) AddUser(ctx context.Context, param *User) error {
	if err := param.Valid(); err != nil {
		return err
	}
	err := uds.repo.Create(ctx, param)
	if err != nil {
		return err
	}
	return nil
}

func (ud *UserDomainService) Delete(ctx context.Context, id string) error {
	ctxValue := ctx.Value("ctxInfo").(pkg.CtxInfo)

	err := ud.repo.Delete(ctx, id)
	if err != nil {
		slog.Error("can not complete DeleteUser Repository", "error msg", err, "request id", ctxValue.RequestId)
		return err
	}

	slog.Info("process done DeleteUser Repository", "request id", ctxValue.RequestId, "user id", id)
	if err != nil {
		slog.Error("can not complete DeleteUser Repository", "error msg", err, "request id", ctxValue.RequestId)
		return err
	}
	slog.Info("process done DeleteUser Repository", "request id", ctxValue.RequestId, "user id", id)
	return nil
}

func (ud *UserDomainService) SaveUser(ctx context.Context, param *User) error {
	ctxValue := ctx.Value("ctxInfo").(pkg.CtxInfo)

	err := ud.repo.Save(ctx, param)
	if err != nil {
		slog.Error("can not complete EditUser Repository", "error msg", err, "request id", ctxValue.RequestId)
		return err
	}

	slog.Info("process done EditUser Repository", "request id", ctxValue.RequestId, "user id", param.GetID())
	return nil
}

func (ud *UserDomainService) UpdateUser(ctx context.Context, param *User) error {
	ctxValue := ctx.Value("ctxInfo").(pkg.CtxInfo)

	err := ud.repo.Update(ctx, param)
	if err != nil {
		slog.Error("can not complete UpdateUser Repository", "error msg", err, "request id", ctxValue.RequestId)
		return err
	}

	slog.Info("process done UpdateUser Repository", "request id", ctxValue.RequestId, "user id", param.GetID())
	return nil
}
