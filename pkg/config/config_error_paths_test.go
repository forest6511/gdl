package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	gdlerrors "github.com/forest6511/gdl/pkg/errors"
)

// TestConfigLoaderLoadErrorPaths tests error handling in Load method
func TestConfigLoaderLoadErrorPaths(t *testing.T) {
	t.Run("InvalidJSONFormat", func(t *testing.T) {
		tempDir := t.TempDir()
		configPath := filepath.Join(tempDir, "invalid.json")

		// Write invalid JSON
		invalidJSON := `{"retry_policy": {"max_retries": "not a number"}}`
		if err := os.WriteFile(configPath, []byte(invalidJSON), 0o600); err != nil {
			t.Fatalf("Failed to write test file: %v", err)
		}

		loader := NewConfigLoader(configPath)
		config, err := loader.Load()

		if err == nil {
			t.Fatal("Expected error for invalid JSON, got nil")
		}
		if config != nil {
			t.Error("Expected nil config for invalid JSON")
		}

		var downloadErr *gdlerrors.DownloadError
		if !gdlerrors.AsDownloadError(err, &downloadErr) {
			t.Fatal("Expected DownloadError")
		}
		if downloadErr.Code != gdlerrors.CodeConfigError {
			t.Errorf("Expected CodeConfigError, got: %s", downloadErr.Code)
		}
		if !strings.Contains(downloadErr.Message, "failed to parse config file") {
			t.Errorf("Expected parse error message, got: %s", downloadErr.Message)
		}
	})

	t.Run("CorruptedJSONFile", func(t *testing.T) {
		tempDir := t.TempDir()
		configPath := filepath.Join(tempDir, "corrupted.json")

		// Write corrupted JSON
		corruptedJSON := `{"retry_policy": {`
		if err := os.WriteFile(configPath, []byte(corruptedJSON), 0o600); err != nil {
			t.Fatalf("Failed to write test file: %v", err)
		}

		loader := NewConfigLoader(configPath)
		config, err := loader.Load()

		if err == nil {
			t.Fatal("Expected error for corrupted JSON, got nil")
		}
		if config != nil {
			t.Error("Expected nil config for corrupted JSON")
		}

		var downloadErr *gdlerrors.DownloadError
		if gdlerrors.AsDownloadError(err, &downloadErr) {
			if downloadErr.Code != gdlerrors.CodeConfigError {
				t.Errorf("Expected CodeConfigError, got: %s", downloadErr.Code)
			}
		}
	})
}

// TestConfigValidateOutputFormatErrors tests validateOutputFormat error paths
func TestConfigValidateOutputFormatErrors(t *testing.T) {
	t.Run("InvalidOutputFormat", func(t *testing.T) {
		config := DefaultConfig()
		config.OutputFormat.Format = "invalid"

		err := config.validateOutputFormat()

		if err == nil {
			t.Fatal("Expected validation error for invalid output format")
		}

		var downloadErr *gdlerrors.DownloadError
		if !gdlerrors.AsDownloadError(err, &downloadErr) {
			t.Fatal("Expected DownloadError")
		}
		if downloadErr.Code != gdlerrors.CodeValidationError {
			t.Errorf("Expected CodeValidationError, got: %s", downloadErr.Code)
		}
		if !strings.Contains(downloadErr.Message, "output_format.format") {
			t.Errorf("Expected field name in message, got: %s", downloadErr.Message)
		}
		if !strings.Contains(downloadErr.Message, "invalid") {
			t.Errorf("Expected 'invalid' in message, got: %s", downloadErr.Message)
		}
	})

	t.Run("InvalidLogLevel", func(t *testing.T) {
		config := DefaultConfig()
		config.OutputFormat.LogLevel = "invalid"

		err := config.validateOutputFormat()

		if err == nil {
			t.Fatal("Expected validation error for invalid log level")
		}

		var downloadErr *gdlerrors.DownloadError
		if !gdlerrors.AsDownloadError(err, &downloadErr) {
			t.Fatal("Expected DownloadError")
		}
		if downloadErr.Code != gdlerrors.CodeValidationError {
			t.Errorf("Expected CodeValidationError, got: %s", downloadErr.Code)
		}
		if !strings.Contains(downloadErr.Message, "output_format.log_level") {
			t.Errorf("Expected field name in message, got: %s", downloadErr.Message)
		}
	})
}

// TestConfigValidateTimeoutsErrors tests validateTimeouts error paths
func TestConfigValidateTimeoutsErrors(t *testing.T) {
	testCases := []struct {
		name          string
		modifyConfig  func(*Config)
		expectedField string
	}{
		{
			name: "NegativeConnectTimeout",
			modifyConfig: func(c *Config) {
				c.Timeouts.ConnectTimeout = -1 * time.Second
			},
			expectedField: "timeouts.connect_timeout",
		},
		{
			name: "ZeroConnectTimeout",
			modifyConfig: func(c *Config) {
				c.Timeouts.ConnectTimeout = 0
			},
			expectedField: "timeouts.connect_timeout",
		},
		{
			name: "NegativeReadTimeout",
			modifyConfig: func(c *Config) {
				c.Timeouts.ReadTimeout = -1 * time.Second
			},
			expectedField: "timeouts.read_timeout",
		},
		{
			name: "NegativeWriteTimeout",
			modifyConfig: func(c *Config) {
				c.Timeouts.WriteTimeout = -1 * time.Second
			},
			expectedField: "timeouts.write_timeout",
		},
		{
			name: "NegativeRequestTimeout",
			modifyConfig: func(c *Config) {
				c.Timeouts.RequestTimeout = -1 * time.Second
			},
			expectedField: "timeouts.request_timeout",
		},
		{
			name: "NegativeDownloadTimeout",
			modifyConfig: func(c *Config) {
				c.Timeouts.DownloadTimeout = -1 * time.Second
			},
			expectedField: "timeouts.download_timeout",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			config := DefaultConfig()
			tc.modifyConfig(config)

			err := config.validateTimeouts()

			if err == nil {
				t.Fatalf("Expected validation error for %s", tc.name)
			}

			var downloadErr *gdlerrors.DownloadError
			if !gdlerrors.AsDownloadError(err, &downloadErr) {
				t.Fatal("Expected DownloadError")
			}
			if downloadErr.Code != gdlerrors.CodeValidationError {
				t.Errorf("Expected CodeValidationError, got: %s", downloadErr.Code)
			}
			if !strings.Contains(downloadErr.Message, tc.expectedField) {
				t.Errorf("Expected field %s in message, got: %s", tc.expectedField, downloadErr.Message)
			}
			if !strings.Contains(downloadErr.Message, "must be positive") {
				t.Errorf("Expected 'must be positive' in message, got: %s", downloadErr.Message)
			}
		})
	}
}

// TestConfigValidateNetworkErrors tests validateNetwork error paths
func TestConfigValidateNetworkErrors(t *testing.T) {
	testCases := []struct {
		name          string
		modifyConfig  func(*Config)
		expectedField string
		expectedMsg   string
	}{
		{
			name: "NegativeMaxConcurrentDownloads",
			modifyConfig: func(c *Config) {
				c.Network.MaxConcurrentDownloads = -1
			},
			expectedField: "network.max_concurrent_downloads",
			expectedMsg:   "must be positive",
		},
		{
			name: "ZeroMaxConcurrentDownloads",
			modifyConfig: func(c *Config) {
				c.Network.MaxConcurrentDownloads = 0
			},
			expectedField: "network.max_concurrent_downloads",
			expectedMsg:   "must be positive",
		},
		{
			name: "NegativeChunkSize",
			modifyConfig: func(c *Config) {
				c.Network.ChunkSize = -1
			},
			expectedField: "network.chunk_size",
			expectedMsg:   "must be positive",
		},
		{
			name: "NegativeBufferSize",
			modifyConfig: func(c *Config) {
				c.Network.BufferSize = -1
			},
			expectedField: "network.buffer_size",
			expectedMsg:   "must be positive",
		},
		{
			name: "NegativeMaxRedirects",
			modifyConfig: func(c *Config) {
				c.Network.MaxRedirects = -1
			},
			expectedField: "network.max_redirects",
			expectedMsg:   "must be non-negative",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			config := DefaultConfig()
			tc.modifyConfig(config)

			err := config.validateNetwork()

			if err == nil {
				t.Fatalf("Expected validation error for %s", tc.name)
			}

			var downloadErr *gdlerrors.DownloadError
			if !gdlerrors.AsDownloadError(err, &downloadErr) {
				t.Fatal("Expected DownloadError")
			}
			if downloadErr.Code != gdlerrors.CodeValidationError {
				t.Errorf("Expected CodeValidationError, got: %s", downloadErr.Code)
			}
			if !strings.Contains(downloadErr.Message, tc.expectedField) {
				t.Errorf("Expected field %s in message, got: %s", tc.expectedField, downloadErr.Message)
			}
			if !strings.Contains(downloadErr.Message, tc.expectedMsg) {
				t.Errorf("Expected '%s' in message, got: %s", tc.expectedMsg, downloadErr.Message)
			}
		})
	}
}

// TestConfigValidateStorageErrors tests validateStorage error paths
func TestConfigValidateStorageErrors(t *testing.T) {
	t.Run("NegativeMinFreeSpace", func(t *testing.T) {
		config := DefaultConfig()
		config.Storage.MinFreeSpace = -1

		err := config.validateStorage()

		if err == nil {
			t.Fatal("Expected validation error for negative min free space")
		}

		var downloadErr *gdlerrors.DownloadError
		if !gdlerrors.AsDownloadError(err, &downloadErr) {
			t.Fatal("Expected DownloadError")
		}
		if downloadErr.Code != gdlerrors.CodeValidationError {
			t.Errorf("Expected CodeValidationError, got: %s", downloadErr.Code)
		}
		if !strings.Contains(downloadErr.Message, "storage.min_free_space") {
			t.Errorf("Expected field name in message, got: %s", downloadErr.Message)
		}
		if !strings.Contains(downloadErr.Message, "must be non-negative") {
			t.Errorf("Expected 'must be non-negative' in message, got: %s", downloadErr.Message)
		}
	})
}

// TestConfigValidateIntegration tests that Validate catches all sub-validation errors
func TestConfigValidateIntegration(t *testing.T) {
	t.Run("InvalidConfigTriggersValidate", func(t *testing.T) {
		config := DefaultConfig()
		config.OutputFormat.Format = "invalid"

		err := config.Validate()

		if err == nil {
			t.Fatal("Expected Validate to catch output format error")
		}

		var downloadErr *gdlerrors.DownloadError
		if gdlerrors.AsDownloadError(err, &downloadErr) {
			if downloadErr.Code != gdlerrors.CodeValidationError {
				t.Errorf("Expected CodeValidationError, got: %s", downloadErr.Code)
			}
		}
	})

	t.Run("TimeoutErrorPropagates", func(t *testing.T) {
		config := DefaultConfig()
		config.Timeouts.ConnectTimeout = -1 * time.Second

		err := config.Validate()

		if err == nil {
			t.Fatal("Expected Validate to catch timeout error")
		}
	})

	t.Run("NetworkErrorPropagates", func(t *testing.T) {
		config := DefaultConfig()
		config.Network.ChunkSize = -1

		err := config.Validate()

		if err == nil {
			t.Fatal("Expected Validate to catch network error")
		}
	})
}

// TestConfigLoaderSaveErrorPaths tests Save method error handling
func TestConfigLoaderSaveErrorPaths(t *testing.T) {
	t.Run("WriteToReadOnlyDirectory", func(t *testing.T) {
		if os.Getuid() == 0 {
			t.Skip("Skipping test when running as root (permissions tests won't work)")
		}

		tempDir := t.TempDir()
		readOnlyDir := filepath.Join(tempDir, "readonly")
		if err := os.Mkdir(readOnlyDir, 0o400); err != nil {
			t.Fatalf("Failed to create read-only directory: %v", err)
		}
		defer os.Chmod(readOnlyDir, 0o700)

		configPath := filepath.Join(readOnlyDir, "config.json")
		loader := NewConfigLoader(configPath)
		config := DefaultConfig()

		err := loader.Save(config)

		if err == nil {
			t.Fatal("Expected error when writing to read-only directory")
		}

		var downloadErr *gdlerrors.DownloadError
		if gdlerrors.AsDownloadError(err, &downloadErr) {
			if downloadErr.Code != gdlerrors.CodeConfigError {
				t.Errorf("Expected CodeConfigError, got: %s", downloadErr.Code)
			}
			if !strings.Contains(downloadErr.Message, "failed to write config file") {
				t.Errorf("Expected write error message, got: %s", downloadErr.Message)
			}
		}
	})
}

// TestConfigLoaderRoundTrip tests Load/Save integration with validation
func TestConfigLoaderRoundTrip(t *testing.T) {
	t.Run("SaveAndLoadPreservesConfig", func(t *testing.T) {
		tempDir := t.TempDir()
		configPath := filepath.Join(tempDir, "roundtrip.json")

		loader := NewConfigLoader(configPath)
		originalConfig := DefaultConfig()
		originalConfig.Network.MaxConcurrentDownloads = 10
		originalConfig.RetryPolicy.MaxRetries = 5

		// Save
		if err := loader.Save(originalConfig); err != nil {
			t.Fatalf("Failed to save config: %v", err)
		}

		// Load
		loadedConfig, err := loader.Load()
		if err != nil {
			t.Fatalf("Failed to load config: %v", err)
		}

		// Verify
		if loadedConfig.Network.MaxConcurrentDownloads != originalConfig.Network.MaxConcurrentDownloads {
			t.Errorf("MaxConcurrentDownloads mismatch: got %d, want %d",
				loadedConfig.Network.MaxConcurrentDownloads, originalConfig.Network.MaxConcurrentDownloads)
		}
		if loadedConfig.RetryPolicy.MaxRetries != originalConfig.RetryPolicy.MaxRetries {
			t.Errorf("MaxRetries mismatch: got %d, want %d",
				loadedConfig.RetryPolicy.MaxRetries, originalConfig.RetryPolicy.MaxRetries)
		}
	})

	t.Run("SavedConfigPassesValidation", func(t *testing.T) {
		tempDir := t.TempDir()
		configPath := filepath.Join(tempDir, "valid.json")

		loader := NewConfigLoader(configPath)
		config := DefaultConfig()

		if err := loader.Save(config); err != nil {
			t.Fatalf("Failed to save config: %v", err)
		}

		loadedConfig, err := loader.Load()
		if err != nil {
			t.Fatalf("Failed to load config: %v", err)
		}

		if err := loadedConfig.Validate(); err != nil {
			t.Errorf("Saved config should pass validation: %v", err)
		}
	})
}

// TestConfigErrorMessages verifies error message quality
func TestConfigErrorMessages(t *testing.T) {
	t.Run("ValidationErrorIncludesValue", func(t *testing.T) {
		config := DefaultConfig()
		config.Timeouts.ConnectTimeout = -5 * time.Second

		err := config.validateTimeouts()

		if err == nil {
			t.Fatal("Expected validation error")
		}

		errorMsg := err.Error()
		if !strings.Contains(errorMsg, "-5s") {
			t.Errorf("Error message should include the invalid value: %s", errorMsg)
		}
	})

	t.Run("LoadErrorIncludesPath", func(t *testing.T) {
		tempDir := t.TempDir()
		configPath := filepath.Join(tempDir, "bad.json")

		if err := os.WriteFile(configPath, []byte("{invalid}"), 0o600); err != nil {
			t.Fatalf("Failed to write test file: %v", err)
		}

		loader := NewConfigLoader(configPath)
		_, err := loader.Load()

		if err == nil {
			t.Fatal("Expected error for invalid JSON")
		}

		var downloadErr *gdlerrors.DownloadError
		if gdlerrors.AsDownloadError(err, &downloadErr) {
			if !strings.Contains(downloadErr.Details, configPath) {
				t.Errorf("Error details should include config path: %s", downloadErr.Details)
			}
		}
	})
}

// TestConfigMarshalability verifies that configs can be serialized
func TestConfigMarshalability(t *testing.T) {
	t.Run("DefaultConfigMarshalable", func(t *testing.T) {
		config := DefaultConfig()

		data, err := json.MarshalIndent(config, "", "  ")
		if err != nil {
			t.Fatalf("Failed to marshal default config: %v", err)
		}

		var unmarshaled Config
		if err := json.Unmarshal(data, &unmarshaled); err != nil {
			t.Fatalf("Failed to unmarshal config: %v", err)
		}

		if err := unmarshaled.Validate(); err != nil {
			t.Errorf("Unmarshaled config should be valid: %v", err)
		}
	})
}
