package middleware

import (
	"github.com/gofiber/fiber/v2"

	"github.com/dock-tech/munin-exchange-rate-api/pkg/ulid"
)

const TRACE_ID = "trace-id"

func TraceIdMiddleware(ctx *fiber.Ctx) error {
	traceId := ctx.Get(TRACE_ID)

	if len(traceId) == 0 {
		traceId = ulid.NewUlid().UUIDString
	}

	ctx.Set(TRACE_ID, traceId)
	ctx.Request().Header.Set(TRACE_ID, traceId)
	ctx.Response().Header.Set(TRACE_ID, traceId)

	return ctx.Next()
}
