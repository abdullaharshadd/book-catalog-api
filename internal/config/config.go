package config

import (
	"github.com/spf13/viper"
)

type Config struct {
	DatabaseURL string `mapstructure:"DATABASE_URL"`
	Port        string `mapstructure:"PORT"`
	JWTSecret   string `mapstructure:"JWT_SECRET"`
}

func Load() (*Config, error) {
	viper.AutomaticEnv()

	viper.SetDefault("DATABASE_URL", "postgres://app:app@db:5432/app?sslmode=disable")
	viper.SetDefault("PORT", "8080")
	viper.SetDefault("JWT_SECRET", "")

	_ = viper.BindEnv("DATABASE_URL", "DATABASE_URL")
	_ = viper.BindEnv("PORT", "PORT")
	_ = viper.BindEnv("JWT_SECRET", "JWT_SECRET")

	cfg := &Config{}
	if err := viper.Unmarshal(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}