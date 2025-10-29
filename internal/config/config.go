package config

import (
	"github.com/ilyakaznacheev/cleanenv"
)

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
