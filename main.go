package main

import (
	"github.com/joho/godotenv"

	"github.com/fsvxavier/default-hexagonal/cmd/webserver"
	logger "github.com/fsvxavier/default-hexagonal/pkg/logger/zap"
)

// @title		template API
// @version	1.0
// @host		localhost:8086
// @schemes	http.
func main2() {
	// To load our environmental variables.
	err := godotenv.Load(".env")
	if err != nil {
		logger.DebugOutCtx("\nNo .env file avaliable seaching ENVIROMENTS system")
	}

	webserver.Run()
}
