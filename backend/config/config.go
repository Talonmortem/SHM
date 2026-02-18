package config

import "os"

type Config struct {
	JWTSecret string
	DBDSN     string
	APIKey    string
}

func Load() Config {
	return Config{
		JWTSecret: os.Getenv("JWT_SECRET"),
		DBDSN:     os.Getenv("DB_DSN"),
		APIKey:    os.Getenv("API_KEY"),
	}
}
