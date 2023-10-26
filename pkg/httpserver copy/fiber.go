package httpserver

import (
	"context"
	"net/http"
	"os"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/monitor"
	"github.com/gofiber/fiber/v2/middleware/pprof"
	"github.com/gofiber/fiber/v2/middleware/skip"
	jsoniter "github.com/json-iterator/go"
	fibertrace "gopkg.in/DataDog/dd-trace-go.v1/contrib/gofiber/fiber.v2"

	"github.com/dock-tech/munin-exchange-rate-api/pkg/httpserver/middleware"
	"github.com/dock-tech/munin-exchange-rate-api/pkg/logger"
)

type WebServer struct {
	App             *fiber.App
	DDServiceName   string
	Host            string
	Port            string
	Network         string
	ReadBufferSize  int
	Concurrency     int32
	Datadog         bool
	Metrics         bool
	Compress        bool
	Prefork         bool
	Rmu             bool
	DisableStartMsg bool
	Pprof           bool
}

const (
	TRUE        = "true"
	FALSE       = "false"
	CONCURRENCY = 262144
	READ_BUFFER = 4096
)

var (
	json            = jsoniter.ConfigCompatibleWithStandardLibrary
	healthcheckPath = func(c *fiber.Ctx) bool { return c.Path() == "/health" }
	pagemetricsPath = func(c *fiber.Ctx) bool { return c.Path() == "/pagemetrics" }
	apimetricsPath  = func(c *fiber.Ctx) bool { return c.Path() == "/apimetrics" }
)

func NewServer(host, port string) *WebServer {
	webs := &WebServer{
		Host:            host,
		Port:            port,
		Datadog:         os.Getenv("DATADOG_ENABLED") == TRUE,
		Pprof:           os.Getenv("PPROF_ENABLED") == TRUE,
		Metrics:         os.Getenv("HTTP_METRICS_ENABLED") == TRUE,
		Compress:        os.Getenv("HTTP_COMPRESS_ENABLED") == TRUE,
		DisableStartMsg: os.Getenv("HTTP_DISABLE_START_MSG") == TRUE,
		Prefork:         os.Getenv("HTTP_PREFORK") == TRUE,
		Concurrency:     CONCURRENCY,
		ReadBufferSize:  READ_BUFFER,
		Rmu:             os.Getenv("HTTP_RMU") == TRUE,
		Network:         os.Getenv("HTTP_NETWORK"),
		DDServiceName:   os.Getenv("DD_SERVICE"),
	}

	webs.App = fiber.New(fiber.Config{
		ErrorHandler:          middleware.ApplicationErrorHandler,
		Prefork:               webs.Prefork,
		JSONEncoder:           json.Marshal,
		JSONDecoder:           json.Unmarshal,
		DisableStartupMessage: webs.DisableStartMsg,
	})

	return webs
}

func (webs *WebServer) SetDDServiceName(serviceName string) *WebServer {
	webs.DDServiceName = serviceName
	return webs
}

func (webs *WebServer) EnableDDService(enabled bool) *WebServer {
	webs.Datadog = enabled
	return webs
}

func (webs *WebServer) EnableMetrics(enabled bool) *WebServer {
	webs.Metrics = enabled
	return webs
}

func (webs *WebServer) Start(ctx context.Context) (err error) {
	log := logger.NewLogger()
	log.WithContext(ctx)

	webs.App.Use(skip.New(middleware.LoggerMiddleware(os.Stdout), healthcheckPath))

	if webs.Datadog {
		webs.App.Use(skip.New(fibertrace.Middleware(fibertrace.WithServiceName(webs.DDServiceName), fibertrace.WithAnalytics(true)), healthcheckPath))
	}

	if webs.Metrics {
		webs.App.Use(skip.New(middleware.LoggerMiddleware(os.Stdout), pagemetricsPath))
		webs.App.Use(skip.New(middleware.LoggerMiddleware(os.Stdout), apimetricsPath))
		webs.App.Use(skip.New(fibertrace.Middleware(fibertrace.WithServiceName(webs.DDServiceName), fibertrace.WithAnalytics(true)), pagemetricsPath))
		webs.App.Use(skip.New(fibertrace.Middleware(fibertrace.WithServiceName(webs.DDServiceName), fibertrace.WithAnalytics(true)), apimetricsPath))
	}

	if webs.Pprof {
		webs.App.Use(pprof.New())
	}

	err = webs.App.Listen("0.0.0.0:" + webs.Port)
	if err != nil {
		log.Error(context.TODO(), "The port is already in use!")
	}
	return err
}

func readyCheck(ctx *fiber.Ctx) error {
	message := map[string]string{"status": "ok"}
	return ctx.JSON(message)
}

func liveCheck(ctx *fiber.Ctx) error {
	message := map[string]string{"status": "ok"}
	return ctx.JSON(message)
}

func (webs *WebServer) DefaultRouter(api *fiber.App) {
	router := api.Group("/")

	router.Get("/health", func(ctx *fiber.Ctx) error {
		logger.Debug(ctx.UserContext(), ctx.Get("X-Kubernetes-Probe"))

		switch ctx.Get("X-Kubernetes-Probe") {
		case "ready":
			return readyCheck(ctx)
		case "live":
			return liveCheck(ctx)
		default:
			return liveCheck(ctx)
		}
	})

	// api.Route("/docs/*", func(r fiber.Router) {
	// 	r.Get("", swagger.New(swagger.Config{
	// 		DocExpansion: "none",
	// 	}))
	// })

	if webs.Metrics {
		router.Get("/pagemetrics", monitor.New(monitor.Config{
			APIOnly: false,
		}))

		router.Get("/apimetrics", monitor.New(monitor.Config{
			APIOnly: true,
		}))
	}

	router.All("/*", func(ctx *fiber.Ctx) error {
		ctx.Status(http.StatusForbidden)
		return ctx.JSON(fiber.Map{"message": "Forbidden"})
	})
}
