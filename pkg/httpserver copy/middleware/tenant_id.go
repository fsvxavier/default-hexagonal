package middleware

import (
	"context"

	"github.com/gofiber/fiber/v2"
)

const (
	CLIENT_ID = "client-id"
	TENANT_ID = "tenant_id"
)

func TenantIdMiddleware(ctx *fiber.Ctx) error {
	tenantID := ctx.Get(CLIENT_ID)
	ctx.Set(TENANT_ID, tenantID)

	ctxs := ctx.UserContext()
	//nolint:staticcheck // SA1029 ignore this!
	ctxs = context.WithValue(ctxs, TENANT_ID, tenantID)
	ctx.SetUserContext(ctxs)

	return ctx.Next()
}
