package cli

import (
	"fmt"
	"log"
	"net"

	"company-tree/internal/config"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}
	host, port, err := net.SplitHostPort(cfg.DB.Addr)
	if err != nil {
		log.Fatal(err)
	}
	dsn := fmt.Sprintf("host=%s port=%s user=%s dbname=%s password=%s sslmode=%s",
		host, port, cfg.DB.User, cfg.DB.Name, cfg.DB.Password, cfg.DB.SSLMode)
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal(err)
	}
	// TODO: как закрыть?
	_ = db
}
