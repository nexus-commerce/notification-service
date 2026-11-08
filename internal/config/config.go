package config

import (
	"net"

	"github.com/caarlos0/env/v6"
	"github.com/ilyakaznacheev/cleanenv"
)

type Kafka struct {
	Host string `env:"KAFKA_HOST,required"`
	Port string `env:"KAFKA_PORT" envDefault:"29092"`
}

func (k *Kafka) Addr() string {
	return net.JoinHostPort(k.Host, k.Port)
}

type Consumer struct {
	GroupID string `env:"KAFKA_CONSUMER_GROUP" envDefault:"notification-service"`
}

type SendGrid struct {
	APIKey      string `env:"SENDGRID_API_KEY,required"`
	FromName    string `env:"MAIL_FROM_NAME" envDefault:"MyEcom"`
	FromAddress string `env:"MAIL_FROM_ADDRESS" envDefault:"contact@my-ecom-project.dynv6.net"`
}

type Notification struct {
	PrivateKey string `env:"NOTIFICATION_PRIVATE_KEY,required"`
	KeyID      string `env:"NOTIFICATION_KEY_ID,required"`
}

type Frontend struct {
	URL string `env:"FRONTEND_URL" envDefault:"http://localhost:5173"`
}

func Load[T any]() (*T, error) {
	cfg := new(T)
	if err := env.Parse(cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

// Config and New are the legacy loader, removed together with the old code.
type Config struct {
	Sendgrid struct {
		APIKey string `env:"SENDGRID_API_KEY" envDefault:"your_sendgrid_api_key"`
	}
	Kafka struct {
		Host string `env:"KAFKA_HOST" envDefault:"kafka"`
		Port string `env:"KAFKA_PORT" envDefault:"9092"`
	}
}

func New() (*Config, error) {
	var config Config

	if err := cleanenv.ReadEnv(&config); err != nil {
		return nil, err
	}

	return &config, nil
}
