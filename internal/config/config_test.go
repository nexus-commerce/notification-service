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
	t.Setenv("MAIL_FROM_ADDRESS", "from@example.com")
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
	if cfg.FromName != "NexusEcommerce" {
		t.Errorf("FromName = %q", cfg.FromName)
	}
	if cfg.Frontend.URL != "http://localhost:5173" {
		t.Errorf("URL = %q", cfg.Frontend.URL)
	}
}

func TestLoadRejectsEmptyRequired(t *testing.T) {
	for _, name := range []string{"SENDGRID_API_KEY", "MAIL_FROM_ADDRESS", "NOTIFICATION_PRIVATE_KEY", "NOTIFICATION_KEY_ID", "KAFKA_HOST"} {
		t.Run(name, func(t *testing.T) {
			setRequired(t)
			t.Setenv(name, "")
			if _, err := Load[testCfg](); err == nil {
				t.Errorf("Load() with empty %s error = nil, want error", name)
			}
		})
	}
}

func TestLoadRequiresUnsetVars(t *testing.T) {
	for _, name := range []string{"KAFKA_HOST", "MAIL_FROM_ADDRESS"} {
		t.Run(name, func(t *testing.T) {
			setRequired(t)
			os.Unsetenv(name)
			if _, err := Load[testCfg](); err == nil {
				t.Errorf("Load() without %s error = nil, want error", name)
			}
		})
	}
}
