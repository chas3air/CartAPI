package config

import (
	"fmt"
	"log"

	"github.com/spf13/viper"
)

type PsqlConfig struct {
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Database string `mapstructure:"database"`
	Sslmode  string `mapstructure:"sslmode"`
}

type HTTPConfig struct {
	Env  string `mapstructure:"env"`
	Port int    `mapstructure:"port"`
}

type Config struct {
	HTTP HTTPConfig `mapstructure:"http"`
	Psql PsqlConfig `mapstructure:"psql_conn"`
}

func MustLoad() *Config {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")

	err := viper.ReadInConfig()
	if err != nil {
		log.Fatalf("Error reading config file, %s", err)
	}

	var cfg Config
	err = viper.Unmarshal(&cfg)
	if err != nil {
		log.Fatalf("Unable to decode into struct, %v", err)
	}

	return &cfg
}

func (c *Config) ConnectionString() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s",
		c.Psql.User, c.Psql.Password, c.Psql.Host, c.Psql.Port, c.Psql.Database, c.Psql.Sslmode)
}
