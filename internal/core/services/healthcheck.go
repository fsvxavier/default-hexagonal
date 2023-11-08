package services

import (
	"context"

	"github.com/gomodule/redigo/redis"
	"github.com/jackc/pgx/v5/pgxpool"

	rep "github.com/fsvxavier/default-hexagonal/internal/adapters/repositories"
	"github.com/fsvxavier/default-hexagonal/internal/core/commons/constants"
	"github.com/fsvxavier/default-hexagonal/internal/core/domains"
	"github.com/fsvxavier/default-hexagonal/internal/core/ports"
)

type healthcheckService struct {
	Db     *pgxpool.Conn
	Redigo ports.IRedigoRepository
}

func NewHealthCheckService(db *pgxpool.Conn, rdbConn redis.Conn) ports.IHealthCheckService {
	return &healthcheckService{
		Db:     db,
		Redigo: rep.NewRedigoRepository(rdbConn),
	}
}

func (hlc *healthcheckService) GetHealthcheck() (healthStatus *domains.HealthCheck, err error) {
	healthStatus = &domains.HealthCheck{
		AppStatus:    constants.OK,
		AppMessage:   "Application ON",
		DbStatus:     constants.OK,
		DbMessage:    "DB ON",
		RedisStatus:  constants.OK,
		RedisMessage: "Redis ON",
	}

	err = hlc.Db.Ping(context.TODO())
	if err != nil {
		healthStatus.DbStatus = constants.ERROR
		healthStatus.DbMessage = err.Error()
	}

	err = hlc.Redigo.Ping(context.TODO())
	if err != nil {
		healthStatus.RedisStatus = constants.ERROR
		healthStatus.RedisMessage = err.Error()
	}

	// Else return notes
	return healthStatus, nil
}
