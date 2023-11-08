package ports

import "github.com/fsvxavier/default-hexagonal/internal/core/domains"

type IHealthCheckService interface {
	GetHealthcheck() (healthStatus *domains.HealthCheck, err error)
}
