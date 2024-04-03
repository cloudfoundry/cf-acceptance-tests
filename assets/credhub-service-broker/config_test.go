package main

import (
	"os"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	os.Setenv("CREDHUB_API", "https://example.com")
	os.Setenv("CREDHUB_CLIENT", "test-client")
	os.Setenv("CREDHUB_SECRET", "test-secret")

	tests := []struct {
		name     string
		setup    func()
		teardown func()

		expectPort        int
		expectServiceName string
		expectPanic       bool
	}{
		{
			name:              "default",
			expectPort:        8080,
			expectServiceName: "credhub-read",
		},
		{
			name: "custom port and service name",
			setup: func() {
				os.Setenv("PORT", "9000")
				os.Setenv("SERVICE_NAME", "my-service")
			},
			teardown: func() {
				os.Unsetenv("PORT")
			},
			expectPort:        9000,
			expectServiceName: "my-service",
		},
		{
			name: "invalid port",
			setup: func() {
				os.Setenv("PORT", "invalid")
			},
			teardown: func() {
				os.Unsetenv("PORT")
			},
			expectPanic: true,
		},
		{
			name: "credhub api not set",
			setup: func() {
				os.Unsetenv("CREDHUB_API")
			},
			teardown: func() {
				os.Setenv("CREDHUB_API", "https://example.com")
			},
			expectPanic: true,
		},
		{
			name: "credhub client not set",
			setup: func() {
				os.Unsetenv("CREDHUB_CLIENT")
			},
			teardown: func() {
				os.Setenv("CREDHUB_CLIENT", "test-client")
			},
			expectPanic: true,
		},
		{
			name: "credhub secret not set",
			setup: func() {
				os.Unsetenv("CREDHUB_SECRET")
			},
			teardown: func() {
				os.Setenv("CREDHUB_SECRET", "test-secret")
			},
			expectPanic: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.setup != nil {
				tc.setup()
			}
			if tc.teardown != nil {
				defer tc.teardown()
			}

			if tc.expectPanic {
				defer func() {
					if r := recover(); r == nil {
						t.Errorf("LoadConfig() should have panicked")
					}
				}()
			}

			got := LoadConfig()
			if got.Port != tc.expectPort {
				t.Errorf("LoadConfig().Port = %d, want %d", got.Port, tc.expectPort)
			}
			if got.ServiceName != tc.expectServiceName {
				t.Errorf("LoadConfig().ServiceName = %s, want %s", got.ServiceName, tc.expectServiceName)
			}
		})
	}
}
