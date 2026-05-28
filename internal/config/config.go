package config

import (
	"log/slog"
	"time"

	"company-tree/internal/getenv"
	"company-tree/internal/logger"
)

type Logger = logger.Config

type DB struct {
	Addr     string
	User     string
	Password string
	Name     string
	SSLMode  string
}

type Server struct {
	Addr         string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
}

type Config struct {
	Logger Logger
	DB     DB
	Server Server
}

func Load() (Config, error) {
	var ge getenv.Getenv
	required := true

	cfg := Config{
		Logger: Logger{
			Level:     ge.LogLevel("LOG_LEVEL", !required, slog.LevelInfo),
			Plaintext: ge.Bool("LOG_PAINTEXT", !required, false),
		},
		DB: DB{
			Addr:     ge.String("DB_ADDR", required, ""),
			User:     ge.String("DB_USER", !required, "postgres"),
			Password: ge.String("DB_PASSWORD", required, ""),
			Name:     ge.String("DB_NAME", !required, "postgres"),
			SSLMode:  ge.String("DB_SSLMODE", !required, ""),
		},
		Server: Server{
			Addr:         ge.String("SERVER_ADDR", required, ""),
			ReadTimeout:  ge.Duration("READ_TIMEOUT", !required, 5*time.Second),
			WriteTimeout: ge.Duration("WRITE_TIMEOUT", !required, 5*time.Second),
		},
	}

	return cfg, ge.Err()
}
