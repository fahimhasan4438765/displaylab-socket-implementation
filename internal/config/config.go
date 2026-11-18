package config

import "os"

type Config struct {
	HTTPAddr     string
	InternalKey  string
	AllowOrigins string
}

func Load() Config {
	return Config{
		HTTPAddr:     getenv("HTTP_ADDR", ":8090"),
		InternalKey:  getenv("INTERNAL_API_KEY", "displaylab-local-socket-key"),
		AllowOrigins: getenv("WS_ALLOW_ORIGINS", "*"),
	}
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
