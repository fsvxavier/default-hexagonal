package middleware

import (
	"io"

	"github.com/gofiber/fiber/v2"
	json "github.com/json-iterator/go"

	"github.com/dock-tech/munin-exchange-rate-api/pkg/logger"
)

func LoggerMiddleware(w io.Writer) func(*fiber.Ctx) (err error) {
	return func(fiberCtx *fiber.Ctx) (err error) {
		// Capture any error returned by the handler
		err = fiberCtx.Next()
		if err != nil {
			return err
		}

		var body map[string]any
		err = json.Unmarshal(fiberCtx.Body(), &body)
		if err != nil {
			return err
		}

		header := make(map[string]string)
		fiberCtx.Request().Header.VisitAll(func(k, v []byte) {
			header[string(k)] = string(v)
		})

		var headers map[string]string
		marshalHeaders, err := json.Marshal(header)
		if err != nil {
			return err
		}

		err = json.Unmarshal(marshalHeaders, &headers)
		if err != nil {
			return err
		}

		if _, ok := headers["Authorization"]; ok {
			headers["Authorization"] = "***"
		}

		fields := map[string]interface{}{
			"headers": headers,
			"body":    body,
			"params":  fiberCtx.AllParams(),
			"path":    fiberCtx.Path(),
			"url":     fiberCtx.BaseURL(),
			"status":  fiberCtx.Response().StatusCode(),
			"method":  fiberCtx.Method(),
		}
		logger.NewLogger().WithFields(fields)

		return err
	}
}
