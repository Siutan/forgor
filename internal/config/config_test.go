package config

import (
	"testing"
)

func TestValidateProfile(t *testing.T) {
	tests := []struct {
		name    string
		profile Profile
		wantErr bool
	}{
		{
			name: "valid openai profile",
			profile: Profile{
				Provider: "openai",
				APIKey:   "test-key",
				Model:    "gpt-4",
			},
			wantErr: false,
		},
		{
			name: "missing provider",
			profile: Profile{
				APIKey: "test-key",
				Model:  "gpt-4",
			},
			wantErr: true,
		},
		{
			name: "missing model",
			profile: Profile{
				Provider: "openai",
				APIKey:   "test-key",
			},
			wantErr: true,
		},
		{
			name: "missing api key for openai",
			profile: Profile{
				Provider: "openai",
				Model:    "gpt-4",
			},
			wantErr: true,
		},
		{
			name: "valid local profile",
			profile: Profile{
				Provider: "local",
				Model:    "llama2",
				Endpoint: "http://localhost:8080",
			},
			wantErr: false,
		},
		{
			name: "missing endpoint for local",
			profile: Profile{
				Provider: "local",
				Model:    "llama2",
			},
			wantErr: true,
		},
		{
			name: "unsupported provider",
			profile: Profile{
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
		config  Config
		wantErr bool
	}{
		{
			name: "valid config",
			config: Config{
				DefaultProfile: "test",
				Profiles: map[string]Profile{
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
			config: Config{
				Profiles: map[string]Profile{
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
			config: Config{
				DefaultProfile: "missing",
				Profiles: map[string]Profile{
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
			config: Config{
				DefaultProfile: "test",
				Profiles: map[string]Profile{
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
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Config.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGetProfile(t *testing.T) {
	config := Config{
		DefaultProfile: "default",
		Profiles: map[string]Profile{
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
	profile, err := config.GetProfile("")
	if err != nil {
		t.Errorf("GetProfile(\"\") returned error: %v", err)
	}
	if profile.Provider != "openai" {
		t.Errorf("Expected default profile provider to be 'openai', got '%s'", profile.Provider)
	}

	// Test getting specific profile
	profile, err = config.GetProfile("test")
	if err != nil {
		t.Errorf("GetProfile(\"test\") returned error: %v", err)
	}
	if profile.Provider != "anthropic" {
		t.Errorf("Expected test profile provider to be 'anthropic', got '%s'", profile.Provider)
	}

	// Test getting non-existent profile
	_, err = config.GetProfile("missing")
	if err == nil {
		t.Error("GetProfile(\"missing\") should have returned an error")
	}
}
