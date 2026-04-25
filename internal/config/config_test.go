package config_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/your-org/your-project/internal/config"
)

func TestLoadFrom(t *testing.T) {
	tests := []struct {
		name    string
		vars    map[string]string
		want    config.Config
		wantErr bool
	}{
		{
			name: "defaults when map is empty",
			vars: map[string]string{},
			want: config.Config{Domain: "localhost", Port: 8080},
		},
		{
			name: "custom PORT",
			vars: map[string]string{"PORT": "9090"},
			want: config.Config{Domain: "localhost", Port: 9090},
		},
		{
			name: "custom DOMAIN",
			vars: map[string]string{"DOMAIN": "example.com"},
			want: config.Config{Domain: "example.com", Port: 8080},
		},
		{
			name: "custom FRONTEND_ORIGIN",
			vars: map[string]string{"FRONTEND_ORIGIN": "https://frontend.example.com"},
			want: config.Config{Domain: "localhost", FrontendOrigin: "https://frontend.example.com", Port: 8080},
		},
		{
			name: "all custom values",
			vars: map[string]string{
				"PORT":            "3000",
				"DOMAIN":          "api.example.com",
				"FRONTEND_ORIGIN": "https://app.example.com",
			},
			want: config.Config{Domain: "api.example.com", FrontendOrigin: "https://app.example.com", Port: 3000},
		},
		{
			name:    "invalid PORT returns error",
			vars:    map[string]string{"PORT": "not-a-number"},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := config.LoadFrom(tc.vars)
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestLoad_noError(t *testing.T) {
	_, err := config.Load()
	require.NoError(t, err)
}
