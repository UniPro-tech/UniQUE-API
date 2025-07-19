package presenter

import (
	"context"

	"github.com/UniPro-tech/UniQUE-API/api/internal/controller/system"
	"github.com/UniPro-tech/UniQUE-API/api/internal/controller/users"
	userDomain "github.com/UniPro-tech/UniQUE-API/api/internal/domain/user"
	"github.com/UniPro-tech/UniQUE-API/api/internal/driver/mysql"
	"github.com/UniPro-tech/UniQUE-API/api/internal/driver/mysql/repository"
	"github.com/UniPro-tech/UniQUE-API/api/internal/middleware"
	userUsecase "github.com/UniPro-tech/UniQUE-API/api/internal/usecase/user"
	"github.com/gin-contrib/requestid"
	"github.com/gin-gonic/gin"
)

const latest = "/v1"

type Server struct{}

func (s *Server) Run(ctx context.Context) error {
	r := gin.Default()

	// ロガーを設定
	logger := middleware.New()
	httpLogger := middleware.RequestLogger(logger)

	// CORS設定関数
	cors := middleware.CORS()

	// ginにCORSを設定する
	r.Use(cors)

	// ginを使用してリクエスト情報を取得する
	r.Use(httpLogger)

	// request idを付与する
	r.Use(requestid.New())

	v1 := r.Group(latest)
	// 死活監視用
	{
		systemHandler := system.NewSystemHandler()
		v1.GET("/health", systemHandler.Health)
	}

	conn := mysql.New(ctx)
	UserDriver := repository.NewUserDriver(conn)
	UserDomainService := userDomain.NewUserDomainService(UserDriver)
	finduser_usecase := userUsecase.NewFindUserUsecase(UserDomainService)
	finduserbyid_usecase := userUsecase.NewFindUserByIdUsecase(UserDomainService)
	{
		userHandler := users.NewUsersHandler(finduser_usecase, finduserbyid_usecase)
		v1.GET("/users", userHandler.ListUser)
	}

	err := r.Run()
	if err != nil {
		return err
	}

	return nil
}

func NewServer() *Server {
	return &Server{}
}
