package testutil

import (
	"os"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/hashicorp/terraform-plugin-testing/config"
	sdkConf "github.com/stackitcloud/stackit-sdk-go/core/config"
)

func TestConvertConfigVariable(t *testing.T) {
	tests := []struct {
		name     string
		variable config.Variable
		want     string
	}{
		{
			name:     "string",
			variable: config.StringVariable("test"),
			want:     "test",
		},
		{
			name:     "bool: true",
			variable: config.BoolVariable(true),
			want:     "true",
		},
		{
			name:     "bool: false",
			variable: config.BoolVariable(false),
			want:     "false",
		},
		{
			name:     "integer",
			variable: config.IntegerVariable(10),
			want:     "10",
		},
		{
			name:     "quoted string",
			variable: config.StringVariable(`instance =~ ".*"`),
			want:     `instance =~ ".*"`,
		},
		{
			name:     "line breaks",
			variable: config.StringVariable(`line \n breaks`),
			want:     `line \n breaks`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ConvertConfigVariable(tt.variable); got != tt.want {
				t.Errorf("ConvertConfigVariable() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConfigBuilderProviderConfig(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		builder *ConfigBuilder
		want    string
	}{
		{
			name:    "defaults",
			builder: NewConfigBuilder(),
			want: `provider "stackit" {
    default_region = "eu01"
    enable_beta_resources = true
}

`,
		},
		{
			name: "region",
			builder: NewConfigBuilder().
				Region("eu02"),
			want: `provider "stackit" {
    default_region = "eu02"
    enable_beta_resources = true
}

`,
		},
		{
			name: "custom endpoints",
			builder: NewConfigBuilder().
				CustomEndpoint(CdnCustomEndpoint, "http://cdn.example.com").
				CustomEndpoint(DnsCustomEndpoint, "http://dns.example.com"),
			want: `provider "stackit" {
    default_region = "eu01"
    enable_beta_resources = true
    cdn_custom_endpoint = "http://cdn.example.com"
    dns_custom_endpoint = "http://dns.example.com"
}

`,
		},
		{
			name: "experiments",
			builder: NewConfigBuilder().
				Experiments(ExperimentIAM, ExperimentNetwork),
			want: `provider "stackit" {
    default_region = "eu01"
    enable_beta_resources = true
    experiments = ["iam", "network"]
}

`,
		},
		{
			name: "token",
			builder: NewConfigBuilder().
				ServiceAccountToken("expected-token"),
			want: `provider "stackit" {
    default_region = "eu01"
    enable_beta_resources = true
    service_account_token = "expected-token"
}

`,
		},
		{
			name: "everything",
			builder: NewConfigBuilder().
				ServiceAccountToken("expected-token").
				Experiments(ExperimentIAM).
				CustomEndpoint(CdnCustomEndpoint, "http://cdn.example.com"),
			want: `provider "stackit" {
    default_region = "eu01"
    enable_beta_resources = true
    experiments = ["iam"]
    service_account_token = "expected-token"
    cdn_custom_endpoint = "http://cdn.example.com"
}

`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := tt.builder.BuildProviderConfig()
			if d := cmp.Diff(got, tt.want); d != "" {
				t.Errorf("ConfigBuilder.BuildProviderConfig() = diff: %s", d)
			}
		})
	}
}

func TestConfigBuilderProviderConfigEnvVar(t *testing.T) {
	os.Setenv(CdnCustomEndpoint.envVarName, "http://expected.example.com") // nolint:errcheck // test would fail
	defer func() {
		err := os.Unsetenv(CdnCustomEndpoint.envVarName)
		if err != nil {
			t.Fatalf("unset env: %v", err)
		}
	}()
	got := NewConfigBuilder().BuildProviderConfig()
	want := `provider "stackit" {
    default_region = "eu01"
    enable_beta_resources = true
    cdn_custom_endpoint = "http://expected.example.com"
}

`
	if d := cmp.Diff(got, want); d != "" {
		t.Errorf("ConfigBuilder.BuildProviderConfig() = diff: %s", d)
	}
}

func TestConfigBuilderClientOptions(t *testing.T) {
	clientEndpoint := CdnCustomEndpoint
	tests := []struct {
		name    string
		builder *ConfigBuilder
		want    sdkConf.Configuration
	}{
		{
			name:    "default",
			builder: NewConfigBuilder(),
			want: sdkConf.Configuration{
				Region: "eu01",
			},
		},
		{
			name: "custom token endpoint",
			builder: NewConfigBuilder().
				CustomEndpoint(TokenCustomEndpoint, "http://token.example.com"),
			want: sdkConf.Configuration{
				TokenCustomUrl: "http://token.example.com",
				Region:         "eu01",
			},
		},
		{
			name: "token",
			builder: NewConfigBuilder().
				ServiceAccountToken("expected-token"),
			want: sdkConf.Configuration{
				Token:  "expected-token",
				Region: "eu01",
			},
		},
		{
			name: "custom service endpoint",
			builder: NewConfigBuilder().
				CustomEndpoint(clientEndpoint, "http://cdn.example.com"),
			want: sdkConf.Configuration{
				Servers: sdkConf.ServerConfigurations{
					{
						URL:         "http://cdn.example.com",
						Description: "User provided URL",
					},
				},
				Region: "eu01",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			opts := tt.builder.BuildClientOptions(CdnCustomEndpoint)
			got := sdkConf.Configuration{}
			for _, opt := range opts {
				err := opt(&got)
				if err != nil {
					t.Fatalf("Config option returned error: %v", err)
				}
			}
			if d := cmp.Diff(got, tt.want, cmpopts.IgnoreUnexported(sdkConf.Configuration{})); d != "" {
				t.Errorf("ConfigBuilder.BuildClientOptions() = diff: %s", d)
			}
		})
	}
}

func TestConfigBuilderClientOptionsEnvVar(t *testing.T) {
	os.Setenv(CdnCustomEndpoint.envVarName, "http://cdn.example.com") // nolint:errcheck // test would fail
	defer func() {
		err := os.Unsetenv(CdnCustomEndpoint.envVarName)
		if err != nil {
			t.Fatalf("unset env: %v", err)
		}
	}()
	opts := NewConfigBuilder().BuildClientOptions(CdnCustomEndpoint)
	got := sdkConf.Configuration{}
	for _, opt := range opts {
		err := opt(&got)
		if err != nil {
			t.Fatalf("Config option returned error: %v", err)
		}
	}
	want := sdkConf.Configuration{
		Servers: sdkConf.ServerConfigurations{
			{
				URL:         "http://cdn.example.com",
				Description: "User provided URL",
			},
		},
		Region: "eu01",
	}
	if d := cmp.Diff(got, want, cmpopts.IgnoreUnexported(sdkConf.Configuration{})); d != "" {
		t.Errorf("ConfigBuilder.BuildClientOptions() = diff: %s", d)
	}
}
