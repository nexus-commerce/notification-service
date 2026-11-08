package config

import (
	"os"
	"testing"
)

type testCfg struct {
	Kafka
	Consumer
	SendGrid
	Notification
	Frontend
}

func setRequired(t *testing.T) {
	t.Helper()
	t.Setenv("KAFKA_HOST", "kafka")
	t.Setenv("SENDGRID_API_KEY", "k")
	t.Setenv("NOTIFICATION_PRIVATE_KEY", "p")
	t.Setenv("NOTIFICATION_KEY_ID", "dev")
}

func TestLoadAppliesDefaults(t *testing.T) {
	setRequired(t)
	cfg, err := Load[testCfg]()
	if err != nil {
		t.Fatal(err)
	}
	if got := cfg.Kafka.Addr(); got != "kafka:29092" {
		t.Errorf("Addr = %q", got)
	}
	if cfg.GroupID != "notification-service" {
		t.Errorf("GroupID = %q", cfg.GroupID)
	}
	if cfg.FromName != "MyEcom" {
		t.Errorf("FromName = %q", cfg.FromName)
	}
	if cfg.Frontend.URL != "http://localhost:5173" {
		t.Errorf("URL = %q", cfg.Frontend.URL)
	}
}

func TestLoadRequiresKafkaHost(t *testing.T) {
	setRequired(t)
	os.Unsetenv("KAFKA_HOST")
	if _, err := Load[testCfg](); err == nil {
		t.Fatal("expected error")
	}
}
