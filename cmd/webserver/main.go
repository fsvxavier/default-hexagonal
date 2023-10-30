package webserver

import (
	"os"

	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer"

	"github.com/fsvxavier/default-hexagonal/pkg/httpserver/nethttp"
)

const app_name = "template-api"

var app_version = os.Getenv("DD_VERSION")

func Run() {
	serverPort := "8099"

	tracer.Start([]tracer.StartOption{
		tracer.WithLogStartup(false),
		tracer.WithEnv(os.Getenv("ENV")),
		tracer.WithService(app_name),
		tracer.WithServiceVersion(app_version),
		tracer.WithTraceEnabled(true),
		tracer.WithRuntimeMetrics(),
	}...)

	app := nethttp.New()
	router := nethttp.NewRouter()

	app.Router = router

	router.Get("/", func(c *nethttp.Context) error {
		c.String("Hello, World!")
		return nil
	})

	app.Run(":" + string(serverPort))
}
