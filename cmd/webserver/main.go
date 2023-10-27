package webserver

import (
	"os"

	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer"

	"github.com/fsvxavier/default-hexagonal/pkg/httpserver/fiber"
)

const app_name = "template-api"

var app_version = os.Getenv("DD_VERSION")

func Run() {
	serverPort := 8080

	tracer.Start([]tracer.StartOption{
		tracer.WithLogStartup(false),
		tracer.WithEnv(os.Getenv("ENV")),
		tracer.WithService(app_name),
		tracer.WithServiceVersion(app_version),
		tracer.WithTraceEnabled(true),
		tracer.WithRuntimeMetrics(),
	}...)

	frameworks := fiber.FiberEngine{}

	frameworks.Run(serverPort)
}
