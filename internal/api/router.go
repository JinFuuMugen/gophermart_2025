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

	rout.Route("/api/user", func(r chi.Router) {
		r.Post("/register", userHandler.RegisterHandler)
		r.Post("/login", userHandler.LoginHandler)

		r.Group(func(protected chi.Router) {
			protected.Use(userHandler.AuthMiddleware)

			// protected.Post("/orders", handlers.UserOrdersHandler)         // TODO: реализовать позже
			// protected.Get("/orders", handlers.GetUserOrdersHandler)       // TODO
			// protected.Get("/balance", handlers.GetUserBalanceHandler)     // TODO
			// protected.Post("/balance/withdraw", handlers.WithdrawHandler) // TODO
			// protected.Get("/withdrawals", handlers.GetWithdrawalsHandler) // TODO
		})
	})

	return rout
}
