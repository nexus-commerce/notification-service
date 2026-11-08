package config

import (
	"net"

	"github.com/caarlos0/env/v6"
)

type Kafka struct {
	Host string `env:"KAFKA_HOST,required,notEmpty"`
	Port string `env:"KAFKA_PORT" envDefault:"29092"`
}

func (k *Kafka) Addr() string {
	return net.JoinHostPort(k.Host, k.Port)
}

type Consumer struct {
	GroupID string `env:"KAFKA_CONSUMER_GROUP" envDefault:"notification-service"`
}

type SendGrid struct {
	APIKey      string `env:"SENDGRID_API_KEY,required,notEmpty"`
	FromName    string `env:"MAIL_FROM_NAME" envDefault:"MyEcom"`
	FromAddress string `env:"MAIL_FROM_ADDRESS,required,notEmpty"`
}

type Notification struct {
	PrivateKey string `env:"NOTIFICATION_PRIVATE_KEY,required,notEmpty"`
	KeyID      string `env:"NOTIFICATION_KEY_ID,required,notEmpty"`
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
