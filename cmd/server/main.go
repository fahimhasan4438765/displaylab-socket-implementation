package main

import (
	"log"

	"github.com/CreateAbit-Designs/displaylab-socket/internal/config"
	"github.com/CreateAbit-Designs/displaylab-socket/internal/hub"
	"github.com/CreateAbit-Designs/displaylab-socket/internal/httpapi"
	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()

	cfg := config.Load()
	h := hub.New()
	srv := httpapi.NewServer(h, cfg.InternalKey, cfg.AllowOrigins)
	r := httpapi.NewRouter(srv)

	log.Printf("displaylab-socket listening on %s", cfg.HTTPAddr)
	log.Printf("ws:       ws://localhost%s/ws?device_token=<fcm_token>", cfg.HTTPAddr)
	log.Printf("notify:   POST http://localhost%s/internal/v1/notify", cfg.HTTPAddr)

	if err := r.Run(cfg.HTTPAddr); err != nil {
		log.Fatal(err)
	}
}
