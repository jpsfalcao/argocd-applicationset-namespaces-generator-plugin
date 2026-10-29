package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		config  *ServerConfig
		wantErr bool
	}{
		{"token", &ServerConfig{ListenToken: "t"}, false},
		{"no token", &ServerConfig{}, true},
		{"blank token", &ServerConfig{ListenToken: "  "}, true},
		{"local without token", &ServerConfig{Local: true}, false},
		{"local with token", &ServerConfig{Local: true, ListenToken: "t"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.config.validate(); (err != nil) != tt.wantErr {
				t.Errorf("validate() = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// Every case here must be rejected before the handler touches Kubernetes.
func TestHandlerRejectsUnauthorized(t *testing.T) {
	tests := []struct {
		name   string
		config *ServerConfig
		header string
	}{
		{"missing header", &ServerConfig{ListenToken: "t"}, ""},
		{"wrong token", &ServerConfig{ListenToken: "t"}, "Bearer wrong"},
		{"token without Bearer", &ServerConfig{ListenToken: "t"}, "t"},
		{"prefix of token", &ServerConfig{ListenToken: "token"}, "Bearer tok"},
		{"no token configured, not local", &ServerConfig{}, "Bearer anything"},
		{"no token configured, no header", &ServerConfig{}, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/v1/getparams.execute", strings.NewReader(`{"input":{"parameters":{}}}`))
			req.Header.Set("Content-Type", "application/json")
			if tt.header != "" {
				req.Header.Set("Authorization", tt.header)
			}
			rec := httptest.NewRecorder()
			tt.config.secretsHandler(context.Background())(rec, req)
			if rec.Code != http.StatusUnauthorized {
				t.Errorf("status %d, want 401", rec.Code)
			}
		})
	}
}
