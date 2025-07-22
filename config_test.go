package beanq

import (
	"strings"
	"testing"
)

func TestNewConfig(t *testing.T) {
	// Define test cases
	tests := []struct {
		name          string
		configPath    string
		configType    string
		configName    string
		expectedField string // ui.root.username
		expectedErr   string
	}{
		{
			name:          "valid config file",
			configPath:    "./",
			configType:    "json",
			configName:    "env",
			expectedField: "rai",
			expectedErr:   "",
		},
		{
			name:          "invalid config path",
			configPath:    "./err_filepath",
			configType:    "json",
			configName:    "env",
			expectedField: "",
			expectedErr:   "configPath cannot be empty",
		},
		{
			name:          "default config type and name",
			configPath:    "./",
			configType:    "",
			configName:    "",
			expectedField: "rai",
			expectedErr:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			// Call NewConfig with test inputs
			cfg, err := NewConfig(tt.configPath, tt.configType, tt.configName)
			// Check error
			if tt.expectedErr == "" {
				if err != nil {
					t.Errorf("Expected no error, got: %v", err)
				}
			} else {
				if err == nil {
					t.Errorf("Expected error containing %q, got nil", tt.expectedErr)
				} else if !strings.Contains(err.Error(), tt.expectedErr) {
					t.Errorf("Expected error containing %q, got: %v", tt.expectedErr, err)
				}
			}

			// Check config content
			if tt.expectedField != "" {
				if cfg == nil {
					t.Fatal("Expected non-nil config")
				} else {
					if tt.expectedField != cfg.UI.Root.UserName {
						t.Errorf("Expected field %q, got: %v", tt.expectedField, cfg.UI.Root.UserName)
					}
				}
			}
		})
	}
}
