package service

import (
	"testing"

	"ulas-service/internal/config"
)

func devTestConfigEntra() *config.AppConfig {
	cfg := *config.Get()
	cfg.EntraTenantID = "tenant"
	cfg.EntraClientID = "client"
	cfg.EntraClientSecret = "secret"
	cfg.EntraRedirectURL = "https://ulas.example.com/api/auth/entra/callback"
	return &cfg
}

func TestNormalizeEntraUsername(t *testing.T) {
	cases := []struct {
		name     string
		upn      string
		email    string
		fallback string
		want     string
	}{
		{name: "upn prefix lowercased", upn: "John.Doe@bank.com", want: "john.doe"},
		{name: "upn wins over email", upn: "jdoe1@bank.com", email: "ignored@bank.com", want: "jdoe1"},
		{name: "email fallback", email: "Jane.Doe@bank.com", want: "jane.doe"},
		{name: "subject fallback", fallback: "oid-string", want: "oid-string"},
		{name: "all empty", want: ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := normalizeEntraUsername(tc.upn, tc.email, tc.fallback); got != tc.want {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestNewEntraService_NilWhenUnconfigured(t *testing.T) {
	if svc := NewEntraService(nil); svc != nil {
		t.Fatalf("expected nil service for nil config, got %+v", svc)
	}
	cfg := devTestConfigEntra()
	cfg.EntraClientSecret = ""
	if svc := NewEntraService(cfg); svc != nil {
		t.Fatalf("expected nil service when secret missing, got %+v", svc)
	}
	cfg = devTestConfigEntra()
	if svc := NewEntraService(cfg); svc == nil || !svc.Configured() {
		t.Fatalf("expected configured service with all fields set, got %+v", svc)
	}
}
