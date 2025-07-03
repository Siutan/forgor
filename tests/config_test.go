package tests

import (
	"forgor/internal/config"
	"testing"
)

func TestValidateProfile(t *testing.T) {
	tests := []struct {
		name    string
		profile config.Profile
		wantErr bool
	}{
		{
			name: "valid openai profile",
			profile: config.Profile{
				Provider: "openai",
				APIKey:   "test-key",
				Model:    "gpt-4",
			},
			wantErr: false,
		},
		{
			name: "missing provider",
			profile: config.Profile{
				APIKey: "test-key",
				Model:  "gpt-4",
			},
			wantErr: true,
		},
		{
			name: "missing model",
			profile: config.Profile{
				Provider: "openai",
				APIKey:   "test-key",
			},
			wantErr: true,
		},
		{
			name: "missing api key for openai",
			profile: config.Profile{
				Provider: "openai",
				Model:    "gpt-4",
			},
			wantErr: true,
		},
		{
			name: "valid local profile",
			profile: config.Profile{
				Provider: "local",
				Model:    "llama2",
				Endpoint: "http://localhost:8080",
			},
			wantErr: false,
		},
		{
			name: "missing endpoint for local",
			profile: config.Profile{
				Provider: "local",
				Model:    "llama2",
			},
			wantErr: true,
		},
		{
			name: "unsupported provider",
			profile: config.Profile{
				Provider: "unsupported",
				Model:    "test",
				APIKey:   "test-key",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.profile.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Profile.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestConfigValidate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     config.Config
		wantErr bool
	}{
		{
			name: "valid config",
			cfg: config.Config{
				DefaultProfile: "test",
				Profiles: map[string]config.Profile{
					"test": {
						Provider: "openai",
						APIKey:   "test-key",
						Model:    "gpt-4",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "missing default profile",
			cfg: config.Config{
				Profiles: map[string]config.Profile{
					"test": {
						Provider: "openai",
						APIKey:   "test-key",
						Model:    "gpt-4",
					},
				},
			},
			wantErr: true,
		},
		{
			name: "default profile not found",
			cfg: config.Config{
				DefaultProfile: "missing",
				Profiles: map[string]config.Profile{
					"test": {
						Provider: "openai",
						APIKey:   "test-key",
						Model:    "gpt-4",
					},
				},
			},
			wantErr: true,
		},
		{
			name: "invalid profile",
			cfg: config.Config{
				DefaultProfile: "test",
				Profiles: map[string]config.Profile{
					"test": {
						Provider: "openai",
						// Missing APIKey and Model
					},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Config.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGetProfile(t *testing.T) {
	cfg := config.Config{
		DefaultProfile: "default",
		Profiles: map[string]config.Profile{
			"default": {
				Provider: "openai",
				APIKey:   "default-key",
				Model:    "gpt-4",
			},
			"test": {
				Provider: "anthropic",
				APIKey:   "test-key",
				Model:    "claude-3",
			},
		},
	}

	// Test getting default profile
	profile, err := cfg.GetProfile("")
	if err != nil {
		t.Errorf("GetProfile(\"\") returned error: %v", err)
	}
	if profile.Provider != "openai" {
		t.Errorf("Expected default profile provider to be 'openai', got '%s'", profile.Provider)
	}

	// Test getting specific profile
	profile, err = cfg.GetProfile("test")
	if err != nil {
		t.Errorf("GetProfile(\"test\") returned error: %v", err)
	}
	if profile.Provider != "anthropic" {
		t.Errorf("Expected test profile provider to be 'anthropic', got '%s'", profile.Provider)
	}

	// Test getting non-existent profile
	_, err = cfg.GetProfile("missing")
	if err == nil {
		t.Error("GetProfile(\"missing\") should have returned an error")
	}
}
