package middleware

import (
	"github.com/gofiber/fiber/v2"

	"github.com/dock-tech/munin-exchange-rate-api/pkg/httpserver/router"
	"github.com/dock-tech/munin-exchange-rate-api/pkg/logger"
)

func ApplicationErrorHandler(ctx *fiber.Ctx, err error) error {
	logger.Error(ctx.UserContext(), err.Error())

	return router.ResponseAdapter(
		ctx.UserContext(),
		ctx,
		router.ResponseErrorAdapter{Error: err},
	)
}
