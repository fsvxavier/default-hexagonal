package middleware

import (
	"github.com/gofiber/fiber/v2"

	"github.com/fsvxavier/default-hexagonal/pkg/ulid"
)

func TraceIdMiddleware(ctx *fiber.Ctx) error {
	traceId := ctx.Get("Trace-Id")

	if len(traceId) < 1 {
		traceId = ulid.NewUlid().UUIDString
	}

	ctx.Set("Trace-Id", traceId)
	ctx.Request().Header.Set("Trace-Id", traceId)
	ctx.Response().Header.Set("Trace-Id", traceId)

	return ctx.Next()
}
