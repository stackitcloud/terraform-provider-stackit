package access_token

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	_ "embed"
	"encoding/json"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/stackitcloud/stackit-sdk-go/core/clients"
	"github.com/stackitcloud/stackit-sdk-go/core/config"
)

//go:embed testdata/service_account.json
var testServiceAccountKey string

func startMockTokenServer() *httptest.Server {
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		resp := clients.TokenResponseBody{
			AccessToken:  "mock_access_token",
			RefreshToken: "mock_refresh_token",
			TokenType:    "Bearer",
			ExpiresIn:    int(time.Now().Add(time.Hour).Unix()),
			Scope:        "mock_scope",
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	})
	return httptest.NewServer(handler)
}

func generatePrivateKey() (string, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return "", err
	}
	privateKeyPEM := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	}
	return string(pem.EncodeToMemory(privateKeyPEM)), nil
}

func writeTempPEMFile(t *testing.T, pemContent string) string {
	t.Helper()

	tmpFile, err := os.CreateTemp("", "stackit_test_private_key_*.pem")
	if err != nil {
		t.Fatal(err)
	}

	if _, err := tmpFile.WriteString(pemContent); err != nil {
		t.Fatal(err)
	}

	if err := tmpFile.Close(); err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		_ = os.Remove(tmpFile.Name())
	})

	return tmpFile.Name()
}

func TestGetAccessToken(t *testing.T) {
	mockServer := startMockTokenServer()
	t.Cleanup(mockServer.Close)

	privateKey, err := generatePrivateKey()
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		description string
		setupEnv    func()
		cleanupEnv  func()
		cfgFactory  func() *config.Configuration
		expectError bool
		expected    string
	}{
		{
			description: "should return token when service account key passed by value",
			cfgFactory: func() *config.Configuration {
				return &config.Configuration{
					ServiceAccountKey: testServiceAccountKey,
					PrivateKey:        privateKey,
					TokenCustomUrl:    mockServer.URL,
				}
			},
			expectError: false,
			expected:    "mock_access_token",
		},
		{
			description: "should return token when service account key is loaded from file path",
			cfgFactory: func() *config.Configuration {
				return &config.Configuration{
					ServiceAccountKeyPath: "testdata/service_account.json",
					PrivateKey:            privateKey,
					TokenCustomUrl:        mockServer.URL,
				}
			},
			expectError: false,
			expected:    "mock_access_token",
		},
		{
			description: "should fail when private key is invalid",
			cfgFactory: func() *config.Configuration {
				return &config.Configuration{
					ServiceAccountKey: "invalid-json",
					PrivateKey:        "invalid-PEM",
					TokenCustomUrl:    mockServer.URL,
				}
			},
			expectError: true,
			expected:    "",
		},
		{
			description: "should return token when service account key is set via env",
			setupEnv: func() {
				_ = os.Setenv("STACKIT_SERVICE_ACCOUNT_KEY", testServiceAccountKey)
			},
			cleanupEnv: func() {
				_ = os.Unsetenv("STACKIT_SERVICE_ACCOUNT_KEY")
			},
			cfgFactory: func() *config.Configuration {
				return &config.Configuration{
					PrivateKey:     privateKey,
					TokenCustomUrl: mockServer.URL,
				}
			},
			expectError: false,
			expected:    "mock_access_token",
		},
		{
			description: "should return token when service account key path is set via env",
			setupEnv: func() {
				_ = os.Setenv("STACKIT_SERVICE_ACCOUNT_KEY_PATH", "testdata/service_account.json")
			},
			cleanupEnv: func() {
				_ = os.Unsetenv("STACKIT_SERVICE_ACCOUNT_KEY_PATH")
			},
			cfgFactory: func() *config.Configuration {
				return &config.Configuration{
					PrivateKey:     privateKey,
					TokenCustomUrl: mockServer.URL,
				}
			},
			expectError: false,
			expected:    "mock_access_token",
		},
		{
			description: "should return token when private key is set via env",
			setupEnv: func() {
				_ = os.Setenv("STACKIT_PRIVATE_KEY", privateKey)
			},
			cleanupEnv: func() {
				_ = os.Unsetenv("STACKIT_PRIVATE_KEY")
			},
			cfgFactory: func() *config.Configuration {
				return &config.Configuration{
					ServiceAccountKey: testServiceAccountKey,
					TokenCustomUrl:    mockServer.URL,
				}
			},
			expectError: false,
			expected:    "mock_access_token",
		},
		{
			description: "should return token when private key path is set via env",
			setupEnv: func() {
				// Write temp file and set env
				tmpFile := writeTempPEMFile(t, privateKey)
				_ = os.Setenv("STACKIT_PRIVATE_KEY_PATH", tmpFile)
			},
			cleanupEnv: func() {
				_ = os.Unsetenv("STACKIT_PRIVATE_KEY_PATH")
			},
			cfgFactory: func() *config.Configuration {
				return &config.Configuration{
					ServiceAccountKey: testServiceAccountKey,
					TokenCustomUrl:    mockServer.URL,
				}
			},
			expectError: false,
			expected:    "mock_access_token",
		},
		{
			description: "should fail when no service account key or private key is set",
			cfgFactory: func() *config.Configuration {
				return &config.Configuration{
					TokenCustomUrl: mockServer.URL,
				}
			},
			expectError: true,
			expected:    "",
		},
		{
			description: "should fail when no service account key or private key is set via env",
			setupEnv: func() {
				_ = os.Unsetenv("STACKIT_SERVICE_ACCOUNT_KEY")
				_ = os.Unsetenv("STACKIT_SERVICE_ACCOUNT_KEY_PATH")
				_ = os.Unsetenv("STACKIT_PRIVATE_KEY")
				_ = os.Unsetenv("STACKIT_PRIVATE_KEY_PATH")
			},
			cleanupEnv: func() {
				// Restore original environment variables
			},
			cfgFactory: func() *config.Configuration {
				return &config.Configuration{
					TokenCustomUrl: mockServer.URL,
				}
			},
			expectError: true,
			expected:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			if tt.setupEnv != nil {
				tt.setupEnv()
			}
			if tt.cleanupEnv != nil {
				defer tt.cleanupEnv()
			}

			cfg := tt.cfgFactory()

			token, err := getAccessToken(cfg)
			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none for test case '%s'", tt.description)
				}
			} else {
				if err != nil {
					t.Errorf("did not expect error but got: %v for test case '%s'", err, tt.description)
				}
				if token != tt.expected {
					t.Errorf("expected token '%s', got '%s' for test case '%s'", tt.expected, token, tt.description)
				}
			}
		})
	}
}
