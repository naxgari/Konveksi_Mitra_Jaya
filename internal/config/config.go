package config

import (
	"errors"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	Addr         string
	DBDSN        string
	CookieSecure bool
}

func Load() (Config, error) {
	_ = godotenv.Load() // baca .env jika ada

	cookieSecure, _ := strconv.ParseBool(getEnv("COOKIE_SECURE", "false"))

	cfg := Config{
		Addr:         getEnv("ADDR", ":8080"),
		DBDSN:        os.Getenv("DB_DSN"),
		CookieSecure: cookieSecure,
	}
	if cfg.DBDSN == "" {
		return cfg, errors.New("DB_DSN belum di-set (cek file .env)")
	}
	return cfg, nil
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
