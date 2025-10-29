package main

import (
	"github.com/sendgrid/sendgrid-go"
	"log"
	"notification-service/internal/config"
	"notification-service/internal/consumer"
	"notification-service/internal/render"
	"notification-service/internal/service"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	// Config
	cfg, err := config.New()
	if err != nil {
		return err
	}

	// Sendgrid client
	client := sendgrid.NewSendClient(cfg.Sendgrid.APIKey)

	// Templates
	templateCache, err := render.NewTemplateCache()
	if err != nil {
		return err
	}

	// Service
	svc := service.New(client, templateCache)

	// Kafka consumer
	cons := consumer.New(svc, cfg)
	cons.Start()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	<-quit

	log.Println("Received shutdown signal, stopping server...")

	cons.Stop()

	log.Println("Application stopped")

	return nil
}
