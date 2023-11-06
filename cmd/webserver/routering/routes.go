package routering

import (
	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/fsvxavier/default-hexagonal/pkg/database/redis"
	logger "github.com/fsvxavier/default-hexagonal/pkg/logger/zap"
)

type Routes struct {
	App   *fiber.App
	Db    *pgxpool.Pool
	Redis *redis.Redigo
}

func NewRoutes(app *fiber.App, db *pgxpool.Pool, rdb *redis.Redigo) *Routes {
	return &Routes{
		App:   app,
		Db:    db,
		Redis: rdb,
	}
}

func (r *Routes) SetupRoutes() {
	router := r.App.Group("/")

	// Health Routes
	r.healthRoutes(router)
}

func (r *Routes) healthRoutes(router fiber.Router) {
	health := router.Group("/")

	health.Get("/health", func(ctx *fiber.Ctx) error {
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
}

func readyCheck(ctx *fiber.Ctx) error {
	message := map[string]string{"status": "ok"}
	return ctx.JSON(message)
}

func liveCheck(ctx *fiber.Ctx) error {
	message := map[string]string{"status": "ok"}
	return ctx.JSON(message)
}
