package api

import (
	"github.com/JinFuuMugen/gophermart-ya/internal/handlers"
	"github.com/JinFuuMugen/gophermart-ya/internal/logger"
	customMiddleware "github.com/JinFuuMugen/gophermart-ya/internal/middleware"
	"github.com/JinFuuMugen/gophermart-ya/internal/storage"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/chi/v5"
)

func InitRouter(db *storage.Database, jwtSecret []byte) *chi.Mux {
	rout := chi.NewRouter()

	rout.Use(logger.LoggerMiddleware)
	rout.Use(middleware.StripSlashes)
	rout.Use(customMiddleware.GzipMiddleware)

	userHandler := &handlers.UserHandler{
		DB:        db,
		JWTSecret: jwtSecret,
	}

	orderHandler := &handlers.OrderHandler{
		DB: db,
	}

	balanceHandler := &handlers.BalanceHandler{
		DB: db,
	}

	rout.Route("/api/user", func(r chi.Router) {
		r.Post("/register", userHandler.RegisterHandler)
		r.Post("/login", userHandler.LoginHandler)

		r.Group(func(protected chi.Router) {
			protected.Use(userHandler.AuthMiddleware)

			protected.Post("/orders", orderHandler.UploadOrder)
			protected.Get("/orders", orderHandler.GetOrders)
			protected.Get("/balance", balanceHandler.GetBalance)
			protected.Post("/balance/withdraw", balanceHandler.Withdraw)
			protected.Get("/withdrawals", balanceHandler.GetWithdrawals)
		})
	})

	return rout
}
