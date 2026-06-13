package config

import (
	"log/slog"
	"os"
	"time"

	"org-tree-api/internal/lib/getenv"
	"org-tree-api/internal/lib/logger"
)

type Logger = logger.Config

type DB struct {
	Addr     string
	User     string
	Password string
	DBName   string
	SSLMode  string
}

type Server struct {
	Addr            string
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	ShutdownTimeout time.Duration
	RequestTimeout  time.Duration
}

type Config struct {
	Logger Logger
	DB     DB
	Server Server
}

const (
	required = true
	optional = false
)

func Load() (Config, error) {
	ge := getenv.New(os.LookupEnv)

	cfg := Config{
		Logger: Logger{
			Level:     ge.LogLevel("LOG_LEVEL", optional, slog.LevelInfo),
			Plaintext: ge.Bool("LOG_PLAINTEXT", optional, false),
		},
		DB: DB{
			Addr:     ge.String("DB_ADDR", required, ""),
			User:     ge.String("DB_USER", optional, "postgres"),
			Password: ge.String("DB_PASSWORD", required, ""),
			DBName:   ge.String("DB_NAME", optional, "postgres"),
			SSLMode:  ge.String("DB_SSLMODE", optional, "disable"),
		},
		Server: Server{
			Addr:            ge.String("SERVER_ADDR", required, ""),
			ReadTimeout:     ge.Duration("SERVER_READ_TIMEOUT", optional, 5*time.Second),
			WriteTimeout:    ge.Duration("SERVER_WRITE_TIMEOUT", optional, 5*time.Second),
			RequestTimeout:  ge.Duration("SERVER_REQUEST_TIMEOUT", optional, 5*time.Second),
			ShutdownTimeout: ge.Duration("SERVER_SHUTDOWN_TIMEOUT", optional, 10*time.Second),
		},
	}

	return cfg, ge.Err()
}
