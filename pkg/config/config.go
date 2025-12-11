package config

import (
	"errors"
	"fmt"

	"github.com/ilyakaznacheev/cleanenv"
)

type PsqlConfig struct {
	User     string `env:"USER"`
	Password string `env:"PASSWORD"`
	Host     string `env:"HOST"`
	Port     int    `env:"PORT"`
	Database string `env:"DATABASE"`
	Sslmode  string `env:"SSLMODE"`
}

type HTTPConfig struct {
	Env  string `env:"ENV" env-default:"local"`
	Port int    `env:"PORT" env-default:"8080"`
}

type Config struct {
	HTTP HTTPConfig `env-prefix:"HTTP_"`
	Psql PsqlConfig `env-prefix:"PSQL_"`
}

func Load() (*Config, error) {
	var cfg Config
	if err := cleanenv.ReadEnv(&cfg); err != nil {
		return nil, errors.New("cannot read env: " + err.Error())
	}
	return &cfg, nil
}

func (c *Config) ConnectionString() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s",
		c.Psql.User,
		c.Psql.Password,
		c.Psql.Host,
		c.Psql.Port,
		c.Psql.Database,
		c.Psql.Sslmode,
	)
}
