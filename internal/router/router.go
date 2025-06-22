package router

import (
	"github.com/antonminaichev/gophermart-loyalty/internal/balance"
	"github.com/antonminaichev/gophermart-loyalty/internal/logger"
	"github.com/antonminaichev/gophermart-loyalty/internal/middleware"
	"github.com/antonminaichev/gophermart-loyalty/internal/order"
	"github.com/antonminaichev/gophermart-loyalty/internal/user"
	"github.com/go-chi/chi/v5"
	chiMiddleware "github.com/go-chi/chi/v5/middleware"
)

func NewRouter(
	userH *user.Handler,
	orderH *order.Handler,
	balanceH *balance.Handler,
	jwtSecret []byte,
	userRepo user.UserRepository,
) chi.Router {
	r := chi.NewRouter()

	r.Use(logger.WithLogging)
	r.Use(chiMiddleware.Recoverer)

	r.Use(middleware.GzipHandler)

	r.Route("/api/user", func(r chi.Router) {
		r.Post("/register", userH.Register)
		r.Post("/login", userH.Login)
	})

	r.Group(func(r chi.Router) {
		r.Use(middleware.JWTMiddleware(jwtSecret, userRepo))

		r.Post("/api/user/orders", orderH.SubmitOrder)
		r.Get("/api/user/orders", orderH.ListOrders)
		r.Get("/api/user/balance", balanceH.GetBalance)
		r.Post("/api/user/balance/withdraw", balanceH.WithdrawBalance)
		r.Get("/api/user/withdrawals", balanceH.ListWithdrawals)
	})

	return r
}
