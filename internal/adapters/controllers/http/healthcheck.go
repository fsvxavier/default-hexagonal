package handlers

import (
	"github.com/gofiber/fiber/v2"
)

// @Summary HealthCheck
// @Description HealthCheck API
// @Success 200
// @Router /healthcheck [get].
func GetHealthcheck(ctx *fiber.Ctx) error {
	// Else return notes
	return ctx.JSON(fiber.Map{"status": "success", "message": "Healthcheck OK"})
}

func GetHealth(ctx *fiber.Ctx) error {
	return ctx.JSON(fiber.Map{"status": "ok"})
}
