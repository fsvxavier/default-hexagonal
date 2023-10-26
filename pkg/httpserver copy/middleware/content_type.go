package middleware

import (
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/samber/lo"

	"github.com/dock-tech/munin-exchange-rate-api/pkg/errorapi"
)

func ContentTypeMiddleware(method string, allowedContentTypes ...string) func(*fiber.Ctx) error {
	return func(ctx *fiber.Ctx) error {
		if method != ctx.Method() {
			return ctx.Next()
		}

		errorResponse := errorapi.NewApiError()

		contentType := ctx.Get("Content-Type")
		if lo.Contains(allowedContentTypes, contentType) {
			return ctx.Next()
		}

		ctx.Status(fiber.StatusUnsupportedMediaType)
		errorResponse.
			SetErrorCode(strconv.Itoa(fiber.StatusUnsupportedMediaType)).
			SetErrorDescription("Unsupported Media Type")

		errorResponse.SetId(ctx.Get(TRACE_ID))
		if errorResponse != nil {
			return &errorResponse.Error
		}

		return ctx.JSON(errorResponse)
	}
}
