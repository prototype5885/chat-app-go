package models

type Config struct {
	Address           string
	Port              string
	SnowflakeWorkerID uint64
	MysqlUser         string
	MysqlPassword     string
	MysqlAddress      string
	MysqlPort         string
	MysqlDatabase     string
}
