package domains

type HealthCheck struct {
	AppStatus    string
	AppMessage   string
	DbStatus     string
	DbMessage    string
	RedisStatus  string
	RedisMessage string
}
