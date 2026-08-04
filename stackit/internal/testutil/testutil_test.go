package testutil

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	sdkConf "github.com/stackitcloud/stackit-sdk-go/core/config"
)

// expectPanic runs fn and fails the test unless it panics with a message
// containing wantSubstring.
func expectPanic(t *testing.T, wantSubstring string, fn func()) {
	t.Helper()
	defer func() {
		r := recover()
		if r == nil {
			t.Fatalf("expected a panic containing %q, got no panic", wantSubstring)
		}
		if msg := fmt.Sprint(r); !strings.Contains(msg, wantSubstring) {
			t.Fatalf("expected a panic containing %q, got %q", wantSubstring, msg)
		}
	}()
	fn()
}

// newTestState builds a minimal *terraform.State with a single root-module
// resource, so resource.TestCheckFunc values (as returned by CheckListAttr)
// can be exercised without running a real acceptance test.
func newTestState(resourceName string, attributes map[string]string) *terraform.State {
	return &terraform.State{
		Modules: []*terraform.ModuleState{
			{
				Path: []string{"root"},
				Resources: map[string]*terraform.ResourceState{
					resourceName: {
						Primary: &terraform.InstanceState{
							Attributes: attributes,
						},
					},
				},
			},
		},
	}
}

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

func TestResolveVariablePath(t *testing.T) {
	t.Run("no path returns the whole marshaled variable unchanged", func(t *testing.T) {
		got := resolveVariablePath("test", config.StringVariable("hello"))
		if want := `"hello"`; string(got) != want {
			t.Errorf("resolveVariablePath() = %s, want %s", got, want)
		}
	})

	t.Run("int segment indexes into a list", func(t *testing.T) {
		v := config.ListVariable(config.StringVariable("a"), config.StringVariable("b"))
		got := resolveVariablePath("test", v, 1)
		if want := `"b"`; string(got) != want {
			t.Errorf("resolveVariablePath() = %s, want %s", got, want)
		}
	})

	t.Run("string segment looks up an object key", func(t *testing.T) {
		v := config.ObjectVariable(map[string]config.Variable{
			"mode": config.StringVariable("ENABLED"),
		})
		got := resolveVariablePath("test", v, "mode")
		if want := `"ENABLED"`; string(got) != want {
			t.Errorf("resolveVariablePath() = %s, want %s", got, want)
		}
	})

	t.Run("multiple segments navigate through nested lists and objects", func(t *testing.T) {
		v := config.ObjectVariable(map[string]config.Variable{
			"rules": config.ListVariable(
				config.ObjectVariable(map[string]config.Variable{"order": config.IntegerVariable(1)}),
				config.ObjectVariable(map[string]config.Variable{"order": config.IntegerVariable(2)}),
			),
		})
		got := resolveVariablePath("test", v, "rules", 1, "order")
		if want := "2"; string(got) != want {
			t.Errorf("resolveVariablePath() = %s, want %s", got, want)
		}
	})

	t.Run("positive index out of range panics", func(t *testing.T) {
		v := config.ListVariable(config.StringVariable("a"))
		expectPanic(t, "index 5 out of range", func() {
			resolveVariablePath("test", v, 5)
		})
	})

	t.Run("negative index panics", func(t *testing.T) {
		v := config.ListVariable(config.StringVariable("a"))
		expectPanic(t, "index -1 out of range", func() {
			resolveVariablePath("test", v, -1)
		})
	})

	t.Run("unknown object key panics", func(t *testing.T) {
		v := config.ObjectVariable(map[string]config.Variable{"mode": config.StringVariable("x")})
		expectPanic(t, `key "missing" not found`, func() {
			resolveVariablePath("test", v, "missing")
		})
	})

	t.Run("int segment against an object panics instead of silently misreading it", func(t *testing.T) {
		v := config.ObjectVariable(map[string]config.Variable{"mode": config.StringVariable("x")})
		expectPanic(t, "value is not a list", func() {
			resolveVariablePath("test", v, 0)
		})
	})

	t.Run("string segment against a list panics instead of silently misreading it", func(t *testing.T) {
		v := config.ListVariable(config.StringVariable("a"))
		expectPanic(t, "value is not an object", func() {
			resolveVariablePath("test", v, "key")
		})
	})

	t.Run("unsupported segment type panics", func(t *testing.T) {
		v := config.StringVariable("a")
		expectPanic(t, "unsupported path segment type", func() {
			resolveVariablePath("test", v, 1.5)
		})
	})
}

func TestConvertConfigVariablePath(t *testing.T) {
	tests := []struct {
		name     string
		variable config.Variable
		path     []any
		want     string
	}{
		{
			name:     "list index",
			variable: config.ListVariable(config.StringVariable("a"), config.StringVariable("b")),
			path:     []any{1},
			want:     "b",
		},
		{
			name: "object key",
			variable: config.ObjectVariable(map[string]config.Variable{
				"mode": config.StringVariable("ENABLED"),
			}),
			path: []any{"mode"},
			want: "ENABLED",
		},
		{
			name:     "integer element inside a list keeps numeric formatting",
			variable: config.ListVariable(config.IntegerVariable(10), config.IntegerVariable(20)),
			path:     []any{1},
			want:     "20",
		},
		{
			name: "nested object, list index and object key combined",
			variable: config.ObjectVariable(map[string]config.Variable{
				"rules": config.ListVariable(
					config.ObjectVariable(map[string]config.Variable{"description": config.StringVariable("first")}),
					config.ObjectVariable(map[string]config.Variable{"description": config.StringVariable("second")}),
				),
			}),
			path: []any{"rules", 1, "description"},
			want: "second",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ConvertConfigVariable(tt.variable, tt.path...); got != tt.want {
				t.Errorf("ConvertConfigVariable() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCheckListAttr(t *testing.T) {
	const resourceName = "stackit_test.foo"

	t.Run("matches a scalar list exactly", func(t *testing.T) {
		v := config.ListVariable(config.StringVariable("1.2.3.4"), config.StringVariable("5.6.7.8"))
		check := CheckListAttr(resourceName, "ip_acl", v)
		state := newTestState(resourceName, map[string]string{
			"ip_acl.#": "2",
			"ip_acl.0": "1.2.3.4",
			"ip_acl.1": "5.6.7.8",
		})
		if err := check(state); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("empty list", func(t *testing.T) {
		check := CheckListAttr(resourceName, "ip_acl", config.ListVariable())
		state := newTestState(resourceName, map[string]string{
			"ip_acl.#": "0",
		})
		if err := check(state); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("path navigates into a nested object before checking the list", func(t *testing.T) {
		v := config.ObjectVariable(map[string]config.Variable{
			"allowed_http_methods": config.ListVariable(config.StringVariable("GET")),
		})
		check := CheckListAttr(resourceName, "config.waf.allowed_http_methods", v, "allowed_http_methods")
		state := newTestState(resourceName, map[string]string{
			"config.waf.allowed_http_methods.#": "1",
			"config.waf.allowed_http_methods.0": "GET",
		})
		if err := check(state); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("fails when the actual element count does not match", func(t *testing.T) {
		v := config.ListVariable(config.StringVariable("a"), config.StringVariable("b"))
		check := CheckListAttr(resourceName, "ip_acl", v)
		state := newTestState(resourceName, map[string]string{
			"ip_acl.#": "1",
			"ip_acl.0": "a",
		})
		if err := check(state); err == nil {
			t.Error("expected an error for a count mismatch, got nil")
		}
	})

	t.Run("fails when an element value does not match", func(t *testing.T) {
		v := config.ListVariable(config.StringVariable("a"), config.StringVariable("b"))
		check := CheckListAttr(resourceName, "ip_acl", v)
		state := newTestState(resourceName, map[string]string{
			"ip_acl.#": "2",
			"ip_acl.0": "a",
			"ip_acl.1": "WRONG",
		})
		if err := check(state); err == nil {
			t.Error("expected an error for a value mismatch, got nil")
		}
	})

	t.Run("recurses into list-of-object elements as <attr>.<i>.<field>", func(t *testing.T) {
		// Terraform flattens object lists to "<attr>.<i>.<field>". CheckListAttr must check fields individually instead of the whole object.
		v := config.ListVariable(
			config.ObjectVariable(map[string]config.Variable{
				"description": config.StringVariable("first"),
				"order":       config.IntegerVariable(1),
			}),
			config.ObjectVariable(map[string]config.Variable{
				"description": config.StringVariable("second"),
				"order":       config.IntegerVariable(2),
			}),
		)
		check := CheckListAttr(resourceName, "rules", v)
		state := newTestState(resourceName, map[string]string{
			"rules.#":             "2",
			"rules.0.description": "first",
			"rules.0.order":       "1",
			"rules.1.description": "second",
			"rules.1.order":       "2",
		})
		if err := check(state); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("fails when a nested field inside a list-of-object element does not match", func(t *testing.T) {
		v := config.ListVariable(
			config.ObjectVariable(map[string]config.Variable{"order": config.IntegerVariable(1)}),
		)
		check := CheckListAttr(resourceName, "rules", v)
		state := newTestState(resourceName, map[string]string{
			"rules.#":       "1",
			"rules.0.order": "WRONG",
		})
		if err := check(state); err == nil {
			t.Error("expected an error for a nested field mismatch, got nil")
		}
	})

	t.Run("recurses through a list nested inside an object element", func(t *testing.T) {
		// A field of a list-of-objects element can itself be a list.
		v := config.ListVariable(
			config.ObjectVariable(map[string]config.Variable{
				"name": config.StringVariable("rule1"),
				"headers": config.ListVariable(
					config.ObjectVariable(map[string]config.Variable{"name": config.StringVariable("a")}),
					config.ObjectVariable(map[string]config.Variable{"name": config.StringVariable("b")}),
				),
			}),
		)
		check := CheckListAttr(resourceName, "rules", v)
		state := newTestState(resourceName, map[string]string{
			"rules.#":                "1",
			"rules.0.name":           "rule1",
			"rules.0.headers.#":      "2",
			"rules.0.headers.0.name": "a",
			"rules.0.headers.1.name": "b",
		})
		if err := check(state); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("recurses into a nested empty-list field of a list-of-object element", func(t *testing.T) {
		// A field of a list-of-objects element can be an empty list, which
		// config.ListVariable() marshals to JSON null rather than [].
		v := config.ListVariable(
			config.ObjectVariable(map[string]config.Variable{
				"name":    config.StringVariable("rule1"),
				"headers": config.ListVariable(),
			}),
		)
		check := CheckListAttr(resourceName, "rules", v)
		state := newTestState(resourceName, map[string]string{
			"rules.#":           "1",
			"rules.0.name":      "rule1",
			"rules.0.headers.#": "0",
		})
		if err := check(state); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("panics when the resolved value is not a list", func(t *testing.T) {
		expectPanic(t, "resolved value is not a list", func() {
			CheckListAttr(resourceName, "ip_acl", config.StringVariable("not-a-list"))
		})
	})

	t.Run("panics when path navigation fails", func(t *testing.T) {
		v := config.ObjectVariable(map[string]config.Variable{"mode": config.StringVariable("x")})
		expectPanic(t, `key "missing" not found`, func() {
			CheckListAttr(resourceName, "config.waf.missing", v, "missing")
		})
	})
}

func TestCheckObjectAttr(t *testing.T) {
	const resourceName = "stackit_test.foo"

	t.Run("matches all scalar fields of an object", func(t *testing.T) {
		v := config.ObjectVariable(map[string]config.Variable{
			"mode":           config.StringVariable("ENABLED"),
			"type":           config.StringVariable("FREE"),
			"paranoia_level": config.StringVariable("L2"),
		})
		check := CheckObjectAttr(resourceName, "config.waf", v)
		state := newTestState(resourceName, map[string]string{
			"config.waf.mode":           "ENABLED",
			"config.waf.type":           "FREE",
			"config.waf.paranoia_level": "L2",
		})
		if err := check(state); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("empty object", func(t *testing.T) {
		check := CheckObjectAttr(resourceName, "config.waf", config.ObjectVariable(map[string]config.Variable{}))
		state := newTestState(resourceName, map[string]string{})
		if err := check(state); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("path navigates into a nested list element before checking the object", func(t *testing.T) {
		v := config.ListVariable(
			config.ObjectVariable(map[string]config.Variable{
				"description": config.StringVariable("first"),
				"order":       config.IntegerVariable(1),
			}),
		)
		check := CheckObjectAttr(resourceName, "rules.0", v, 0)
		state := newTestState(resourceName, map[string]string{
			"rules.0.description": "first",
			"rules.0.order":       "1",
		})
		if err := check(state); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("recurses into a nested list field as <attr>.<field>.<i>", func(t *testing.T) {
		v := config.ObjectVariable(map[string]config.Variable{
			"mode":                 config.StringVariable("ENABLED"),
			"allowed_http_methods": config.ListVariable(config.StringVariable("GET"), config.StringVariable("POST")),
		})
		check := CheckObjectAttr(resourceName, "config.waf", v)
		state := newTestState(resourceName, map[string]string{
			"config.waf.mode":                   "ENABLED",
			"config.waf.allowed_http_methods.#": "2",
			"config.waf.allowed_http_methods.0": "GET",
			"config.waf.allowed_http_methods.1": "POST",
		})
		if err := check(state); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("recurses into a nested empty-list field", func(t *testing.T) {
		// A field can be an empty list, which config.ListVariable() marshals
		// to JSON null rather than [].
		v := config.ObjectVariable(map[string]config.Variable{
			"mode":              config.StringVariable("ENABLED"),
			"disabled_rule_ids": config.ListVariable(),
		})
		check := CheckObjectAttr(resourceName, "config.waf", v)
		state := newTestState(resourceName, map[string]string{
			"config.waf.mode":                "ENABLED",
			"config.waf.disabled_rule_ids.#": "0",
		})
		if err := check(state); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("fails when a field value does not match", func(t *testing.T) {
		v := config.ObjectVariable(map[string]config.Variable{"mode": config.StringVariable("ENABLED")})
		check := CheckObjectAttr(resourceName, "config.waf", v)
		state := newTestState(resourceName, map[string]string{
			"config.waf.mode": "WRONG",
		})
		if err := check(state); err == nil {
			t.Error("expected an error for a value mismatch, got nil")
		}
	})

	t.Run("panics when the resolved value is not an object", func(t *testing.T) {
		expectPanic(t, "resolved value is not an object", func() {
			CheckObjectAttr(resourceName, "config.waf", config.StringVariable("not-an-object"))
		})
	})

	t.Run("panics when path navigation fails", func(t *testing.T) {
		v := config.ObjectVariable(map[string]config.Variable{"mode": config.StringVariable("x")})
		expectPanic(t, `key "missing" not found`, func() {
			CheckObjectAttr(resourceName, "config.waf", v, "missing")
		})
	})

	t.Run("top-level nil object variable defaults to empty object", func(t *testing.T) {
		// Falls deine Implementierung null zu {} umwandelt:
		check := CheckObjectAttr(resourceName, "config.waf", config.ObjectVariable(nil))
		state := newTestState(resourceName, map[string]string{})
		if err := check(state); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})
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
    enable_beta_resources = false
}`,
		},
		{
			name: "region",
			builder: NewConfigBuilder().
				Region("eu02"),
			want: `provider "stackit" {
    default_region = "eu02"
    enable_beta_resources = false
}`,
		},
		{
			name: "custom endpoints",
			builder: NewConfigBuilder().
				CustomEndpoint(CdnCustomEndpoint, "http://cdn.example.com").
				CustomEndpoint(DnsCustomEndpoint, "http://dns.example.com"),
			want: `provider "stackit" {
    default_region = "eu01"
    enable_beta_resources = false
    cdn_custom_endpoint = "http://cdn.example.com"
    dns_custom_endpoint = "http://dns.example.com"
}`,
		},
		{
			name: "experiments",
			builder: NewConfigBuilder().
				Experiments(ExperimentIAM, ExperimentNetwork, ExperimentSKE),
			want: `provider "stackit" {
    default_region = "eu01"
    enable_beta_resources = false
    experiments = ["iam", "network", "ske"]
}`,
		},
		{
			name: "token",
			builder: NewConfigBuilder().
				ServiceAccountToken("expected-token"),
			want: `provider "stackit" {
    default_region = "eu01"
    enable_beta_resources = false
    service_account_token = "expected-token"
}`,
		},
		{
			name: "everything",
			builder: NewConfigBuilder().
				ServiceAccountToken("expected-token").
				Experiments(ExperimentIAM).
				CustomEndpoint(CdnCustomEndpoint, "http://cdn.example.com"),
			want: `provider "stackit" {
    default_region = "eu01"
    enable_beta_resources = false
    experiments = ["iam"]
    service_account_token = "expected-token"
    cdn_custom_endpoint = "http://cdn.example.com"
}`,
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
    enable_beta_resources = false
    cdn_custom_endpoint = "http://expected.example.com"
}`
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
			want:    sdkConf.Configuration{},
		},
		{
			name: "custom token endpoint",
			builder: NewConfigBuilder().
				CustomEndpoint(TokenCustomEndpoint, "http://token.example.com"),
			want: sdkConf.Configuration{ //nolint:gosec // no hardcoded credentials, just for testcases
				TokenCustomUrl: "http://token.example.com",
			},
		},
		{
			name: "token",
			builder: NewConfigBuilder().
				ServiceAccountToken("expected-token"),
			want: sdkConf.Configuration{
				Token: "expected-token",
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
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			opts := tt.builder.BuildClientOptions(CdnCustomEndpoint, false)
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
	opts := NewConfigBuilder().BuildClientOptions(CdnCustomEndpoint, false)
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
	}
	if d := cmp.Diff(got, want, cmpopts.IgnoreUnexported(sdkConf.Configuration{})); d != "" {
		t.Errorf("ConfigBuilder.BuildClientOptions() = diff: %s", d)
	}
}
