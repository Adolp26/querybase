package models

type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	Redis    RedisConfig    `mapstructure:"redis"`
	Oracle   OracleConfig   `mapstructure:"oracle"`
	Postgres PostgresConfig `mapstructure:"postgres"`
}

type ServerConfig struct {
	Port string `mapstructure:"port"`
	Mode string `mapstructure:"mode"`
}

type RedisConfig struct {
	Host     string `mapstructure:"host"`
	Port     string `mapstructure:"port"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
	TTL      int    `mapstructure:"ttl"`
}

type OracleConfig struct {
	Host     string `mapstructure:"host"`
	Port     string `mapstructure:"port"`
	Service  string `mapstructure:"service"`
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
}

type PostgresConfig struct {
	Host     string `mapstructure:"host"`
	Port     string `mapstructure:"port"`
	Database string `mapstructure:"database"`
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
	SSLMode  string `mapstructure:"sslmode"`
}
