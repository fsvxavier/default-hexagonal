package fiber

import (
	"fmt"
	"net/http"
	"os"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	"github.com/gofiber/fiber/v2/middleware/monitor"
	"github.com/gofiber/fiber/v2/middleware/pprof"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/fiber/v2/middleware/skip"
	"github.com/gofiber/swagger"
	fibertrace "gopkg.in/DataDog/dd-trace-go.v1/contrib/gofiber/fiber.v2"

	"github.com/fsvxavier/default-hexagonal/pkg/httpserver/fiber/middleware"
	log "github.com/fsvxavier/default-hexagonal/pkg/logger/zap"
)

type FiberEngine struct{}

var healthcheckPath = func(c *fiber.Ctx) bool { return c.Path() == "/health" }

func (engine FiberEngine) Run(serverPort string) {
	api := fiber.New(fiber.Config{
		ErrorHandler: middleware.ApplicationErrorHandler,
		// ReadBufferSize:        40960,
		DisableStartupMessage: false,
	})

	if os.Getenv("PPROF_ENABLED") == "true" {
		api.Use(pprof.New())
		api.Get("/metrics", monitor.New())
	}

	api.Use(skip.New(fibertrace.Middleware(), healthcheckPath))

	api.Use(recover.New(recover.Config{
		EnableStackTrace: os.Getenv("SHOW_STACK_TRACE") == "true",
	}))

	api.Use(skip.New(middleware.LoggerMiddleware(os.Stdout), healthcheckPath))
	api.Use(skip.New(middleware.TraceIdMiddleware, healthcheckPath))
	api.Use(skip.New(middleware.TenantIdMiddleware, healthcheckPath))
	api.Use(middleware.ContentTypeMiddleware("POST", fiber.MIMEApplicationJSON))

	if os.Getenv("HTTP_RATE_LIMIT_ENABLE") == "true" {
		api.Use(limiter.New(middleware.DefaultRateLimiterConfig))
	}

	engine.Router(api)

	closed := make(chan bool, 1)

	log.Debugln(fmt.Sprintf("Listening on port %s", serverPort))

	api.Listen(fmt.Sprintf(":%s", serverPort))
	<-closed
}

func liveCheck(ctx *fiber.Ctx) error {
	message := map[string]string{"status": "ok"}
	return ctx.JSON(message)
}

func (FiberEngine) Router(api *fiber.App) {
	api.Get("/health", func(ctx *fiber.Ctx) error {
		log.Debugln(ctx.Path(), ctx.Get("X-Kubernetes-Probe"))

		switch ctx.Get("X-Kubernetes-Probe") {
		case "live":
			return liveCheck(ctx)
		default:
			return liveCheck(ctx)
		}
	})

	api.Route("/docs/*", func(r fiber.Router) {
		r.Get("", swagger.New(swagger.Config{
			DocExpansion: "none",
		}))
	})

	api.All("/*", func(ctx *fiber.Ctx) error {
		ctx.Status(http.StatusForbidden)
		return ctx.JSON(fiber.Map{"message": "Forbidden"})
	})
}
