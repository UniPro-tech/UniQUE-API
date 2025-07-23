package config

import (
	"github.com/caarlos0/env/v6"
)

type Config struct {
	Env          string   `env:"ENV" envDefault:"dev"`
	Database_url string   `env:"DATABASE_URL" envDefult:""`
	ProjectID    string   `env:"PROJECTID" envDefault:""`
	SecretKey    string   `env:"SECRET" envDefault:""`
	FrontendURLs []string `env:"FRONTEND_URLS" envDefault:"http://localhost:3000"`
}

func New() (*Config, error) {
	cfg := &Config{}
	if err := env.Parse(cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}
