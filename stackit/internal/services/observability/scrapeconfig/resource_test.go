package observability

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	observabilitySdk "github.com/stackitcloud/stackit-sdk-go/services/observability/v1api"
)

func TestMapFields(t *testing.T) {
	tests := []struct {
		description string
		input       *observabilitySdk.Job
		expected    Model
		isValid     bool
	}{
		{
			"default_ok",
			&observabilitySdk.Job{
				JobName: "name",
			},
			Model{
				Id:             types.StringValue("pid,iid,name"),
				ProjectId:      types.StringValue("pid"),
				InstanceId:     types.StringValue("iid"),
				Name:           types.StringValue("name"),
				MetricsPath:    types.StringNull(),
				Scheme:         types.StringValue(""),
				ScrapeInterval: types.StringValue(""),
				ScrapeTimeout:  types.StringValue(""),
				SAML2:          types.ObjectNull(saml2Types),
				BasicAuth:      types.ObjectNull(basicAuthTypes),
				Targets:        types.ListNull(types.ObjectType{AttrTypes: targetTypes}),
			},
			true,
		},
		{
			description: "values_ok",
			input: &observabilitySdk.Job{
				JobName:     "name",
				MetricsPath: new("/m"),
				BasicAuth: &observabilitySdk.BasicAuth{
					Password: "p",
					Username: "u",
				},
				Params:         &map[string][]string{"saml2": {"disabled"}, "x": {"y", "z"}},
				Scheme:         observabilitySdk.SCHEME_HTTP.Ptr(),
				ScrapeInterval: "1",
				ScrapeTimeout:  "2",
				SampleLimit:    new(int32(17)),
				StaticConfigs: []observabilitySdk.StaticConfigs{
					{
						Labels:  &map[string]string{"k1": "v1"},
						Targets: []string{"url1"},
					},
					{
						Labels:  &map[string]string{"k2": "v2", "k3": "v3"},
						Targets: []string{"url1", "url3"},
					},
					{
						Labels:  nil,
						Targets: []string{},
					},
				},
			},
			expected: Model{
				Id:             types.StringValue("pid,iid,name"),
				ProjectId:      types.StringValue("pid"),
				InstanceId:     types.StringValue("iid"),
				Name:           types.StringValue("name"),
				MetricsPath:    types.StringValue("/m"),
				Scheme:         types.StringValue("http"),
				ScrapeInterval: types.StringValue("1"),
				ScrapeTimeout:  types.StringValue("2"),
				SampleLimit:    types.Int32Value(17),
				SAML2: types.ObjectValueMust(saml2Types, map[string]attr.Value{
					"enable_url_parameters": types.BoolValue(false),
				}),
				BasicAuth: types.ObjectValueMust(basicAuthTypes, map[string]attr.Value{
					"username": types.StringValue("u"),
					"password": types.StringValue("p"),
				}),
				Targets: types.ListValueMust(types.ObjectType{AttrTypes: targetTypes}, []attr.Value{
					types.ObjectValueMust(targetTypes, map[string]attr.Value{
						"urls": types.ListValueMust(types.StringType, []attr.Value{types.StringValue("url1")}),
						"labels": types.MapValueMust(types.StringType, map[string]attr.Value{
							"k1": types.StringValue("v1"),
						}),
					}),
					types.ObjectValueMust(targetTypes, map[string]attr.Value{
						"urls": types.ListValueMust(types.StringType, []attr.Value{types.StringValue("url1"), types.StringValue("url3")}),
						"labels": types.MapValueMust(types.StringType, map[string]attr.Value{
							"k2": types.StringValue("v2"),
							"k3": types.StringValue("v3"),
						}),
					}),
					types.ObjectValueMust(targetTypes, map[string]attr.Value{
						"urls":   types.ListValueMust(types.StringType, []attr.Value{}),
						"labels": types.MapNull(types.StringType),
					}),
				}),
			},
			isValid: true,
		},
		{
			"response_nil_fail",
			nil,
			Model{},
			false,
		},
		{
			"no_resource_id",
			&observabilitySdk.Job{},
			Model{},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			state := &Model{
				ProjectId:  tt.expected.ProjectId,
				InstanceId: tt.expected.InstanceId,
			}
			err := mapFields(context.Background(), tt.input, state)
			if !tt.isValid && err == nil {
				t.Fatalf("Should have failed")
			}
			if tt.isValid && err != nil {
				t.Fatalf("Should not have failed: %v", err)
			}
			if tt.isValid {
				diff := cmp.Diff(state, &tt.expected)
				if diff != "" {
					t.Fatalf("Data does not match: %s", diff)
				}
			}
		})
	}
}

func TestToCreatePayload(t *testing.T) {
	tests := []struct {
		description    string
		input          *Model
		inputSAML2     *saml2Model
		inputBasicAuth *basicAuthModel
		inputTargets   []targetModel
		expected       *observabilitySdk.CreateScrapeConfigPayload
		isValid        bool
	}{
		{
			"basic_ok",
			&Model{
				MetricsPath: types.StringValue("/metrics"),
			},
			&saml2Model{},
			&basicAuthModel{},
			[]targetModel{},
			&observabilitySdk.CreateScrapeConfigPayload{
				MetricsPath: new("/metrics"),
				// Defaults
				Scheme:         "http",
				ScrapeInterval: "5m",
				ScrapeTimeout:  "2m",
				SampleLimit:    new(float32(5000)),
				StaticConfigs:  []observabilitySdk.CreateScrapeConfigPayloadStaticConfigsInner{},
				Params:         map[string]any{"saml2": []string{"enabled"}},
			},
			true,
		},
		{
			"ok - false enable_url_parameters",
			&Model{
				MetricsPath: types.StringValue("/metrics"),
				Name:        types.StringValue("Name"),
			},
			&saml2Model{
				EnableURLParameters: types.BoolValue(false),
			},
			&basicAuthModel{},
			[]targetModel{},
			&observabilitySdk.CreateScrapeConfigPayload{
				MetricsPath: new("/metrics"),
				JobName:     "Name",
				Params:      map[string]any{"saml2": []string{"disabled"}},
				// Defaults
				Scheme:         "http",
				ScrapeInterval: "5m",
				ScrapeTimeout:  "2m",
				SampleLimit:    new(float32(5000)),
				StaticConfigs:  []observabilitySdk.CreateScrapeConfigPayloadStaticConfigsInner{},
			},
			true,
		},
		{
			"ok -  true enable_url_parameters",
			&Model{
				MetricsPath: types.StringValue("/metrics"),
				Name:        types.StringValue("Name"),
			},
			&saml2Model{
				EnableURLParameters: types.BoolValue(true),
			},
			&basicAuthModel{},
			[]targetModel{},
			&observabilitySdk.CreateScrapeConfigPayload{
				MetricsPath: new("/metrics"),
				JobName:     "Name",
				Params:      map[string]any{"saml2": []string{"enabled"}},
				// Defaults
				Scheme:         "http",
				ScrapeInterval: "5m",
				ScrapeTimeout:  "2m",
				SampleLimit:    new(float32(5000)),
				StaticConfigs:  []observabilitySdk.CreateScrapeConfigPayloadStaticConfigsInner{},
			},
			true,
		},
		{
			"ok - with basic auth",
			&Model{
				MetricsPath: types.StringValue("/metrics"),
				Name:        types.StringValue("Name"),
			},
			&saml2Model{},
			&basicAuthModel{
				Username: types.StringValue("u"),
				Password: types.StringValue("p"),
			},
			[]targetModel{},
			&observabilitySdk.CreateScrapeConfigPayload{
				MetricsPath: new("/metrics"),
				JobName:     "Name",
				BasicAuth: &observabilitySdk.CreateScrapeConfigPayloadBasicAuth{
					Username: new("u"),
					Password: new("p"),
				},
				// Defaults
				Scheme:         "http",
				ScrapeInterval: "5m",
				ScrapeTimeout:  "2m",
				SampleLimit:    new(float32(5000)),
				StaticConfigs:  []observabilitySdk.CreateScrapeConfigPayloadStaticConfigsInner{},
				Params:         map[string]any{"saml2": []string{"enabled"}},
			},
			true,
		},
		{
			"ok - with targets",
			&Model{
				MetricsPath: types.StringValue("/metrics"),
				Name:        types.StringValue("Name"),
			},
			&saml2Model{},
			&basicAuthModel{},
			[]targetModel{
				{
					URLs:   types.ListValueMust(types.StringType, []attr.Value{types.StringValue("url1")}),
					Labels: types.MapValueMust(types.StringType, map[string]attr.Value{"k1": types.StringValue("v1")}),
				},
				{
					URLs:   types.ListValueMust(types.StringType, []attr.Value{types.StringValue("url1"), types.StringValue("url3")}),
					Labels: types.MapValueMust(types.StringType, map[string]attr.Value{"k2": types.StringValue("v2"), "k3": types.StringValue("v3")}),
				},
				{
					URLs:   types.ListValueMust(types.StringType, []attr.Value{}),
					Labels: types.MapNull(types.StringType),
				},
			},
			&observabilitySdk.CreateScrapeConfigPayload{
				MetricsPath: new("/metrics"),
				JobName:     "Name",
				StaticConfigs: []observabilitySdk.CreateScrapeConfigPayloadStaticConfigsInner{
					{
						Targets: []string{"url1"},
						Labels:  map[string]any{"k1": "v1"},
					},
					{
						Targets: []string{"url1", "url3"},
						Labels:  map[string]any{"k2": "v2", "k3": "v3"},
					},
					{
						Targets: []string{},
						Labels:  map[string]any{},
					},
				},
				// Defaults
				Scheme:         "http",
				ScrapeInterval: "5m",
				ScrapeTimeout:  "2m",
				SampleLimit:    new(float32(5000)),
				Params:         map[string]any{"saml2": []string{"enabled"}},
			},
			true,
		},
		{
			"nil_model",
			nil,
			nil,
			nil,
			nil,
			nil,
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			output, err := toCreatePayload(context.Background(), tt.input, tt.inputSAML2, tt.inputBasicAuth, tt.inputTargets)
			if !tt.isValid && err == nil {
				t.Fatalf("Should have failed")
			}
			if tt.isValid && err != nil {
				t.Fatalf("Should not have failed: %v", err)
			}
			if tt.isValid {
				diff := cmp.Diff(output, tt.expected)
				if diff != "" {
					t.Fatalf("Data does not match: %s", diff)
				}
			}
		})
	}
}

func TestToUpdatePayload(t *testing.T) {
	tests := []struct {
		description    string
		input          *Model
		inputSAML2     *saml2Model
		basicAuthModel *basicAuthModel
		inputTargets   []targetModel
		expected       *observabilitySdk.UpdateScrapeConfigPayload
		isValid        bool
	}{
		{
			"basic_ok",
			&Model{
				MetricsPath: types.StringValue("/metrics"),
			},
			&saml2Model{},
			&basicAuthModel{},
			[]targetModel{},
			&observabilitySdk.UpdateScrapeConfigPayload{
				MetricsPath: "/metrics",
				// Defaults
				Scheme:         "http",
				ScrapeInterval: "5m",
				ScrapeTimeout:  "2m",
				SampleLimit:    new(float32(5000)),
				StaticConfigs:  []observabilitySdk.UpdateScrapeConfigPayloadStaticConfigsInner{},
			},
			true,
		},
		{
			"ok -  true enable_url_parameters",
			&Model{
				MetricsPath: types.StringValue("/metrics"),
				Scheme:      types.StringValue("http"),
			},
			&saml2Model{
				EnableURLParameters: types.BoolValue(true),
			},
			&basicAuthModel{},
			[]targetModel{},
			&observabilitySdk.UpdateScrapeConfigPayload{
				MetricsPath: "/metrics",
				// Defaults
				Scheme:         "http",
				ScrapeInterval: "5m",
				ScrapeTimeout:  "2m",
				SampleLimit:    new(float32(5000)),
				StaticConfigs:  []observabilitySdk.UpdateScrapeConfigPayloadStaticConfigsInner{},
				Params:         map[string]any{"saml2": []string{"enabled"}},
			},
			true,
		},
		{
			"ok -  false enable_url_parameters",
			&Model{
				MetricsPath: types.StringValue("/metrics"),
				Scheme:      types.StringValue("http"),
			},
			&saml2Model{
				EnableURLParameters: types.BoolValue(false),
			},
			&basicAuthModel{},
			[]targetModel{},
			&observabilitySdk.UpdateScrapeConfigPayload{
				MetricsPath: "/metrics",
				// Defaults
				Scheme:         "http",
				ScrapeInterval: "5m",
				ScrapeTimeout:  "2m",
				SampleLimit:    new(float32(5000)),
				StaticConfigs:  []observabilitySdk.UpdateScrapeConfigPayloadStaticConfigsInner{},
				Params:         map[string]any{"saml2": []string{"disabled"}},
			},
			true,
		},
		{
			"ok - with basic auth",
			&Model{
				MetricsPath: types.StringValue("/metrics"),
				Name:        types.StringValue("Name"),
			},
			&saml2Model{},
			&basicAuthModel{
				Username: types.StringValue("u"),
				Password: types.StringValue("p"),
			},
			[]targetModel{},
			&observabilitySdk.UpdateScrapeConfigPayload{
				MetricsPath: "/metrics",
				BasicAuth: &observabilitySdk.UpdateScrapeConfigPayloadBasicAuth{
					Username: new("u"),
					Password: new("p"),
				},
				// Defaults
				Scheme:         "http",
				ScrapeInterval: "5m",
				ScrapeTimeout:  "2m",
				SampleLimit:    new(float32(5000)),
				StaticConfigs:  []observabilitySdk.UpdateScrapeConfigPayloadStaticConfigsInner{},
			},
			true,
		},
		{
			"ok - with targets",
			&Model{
				MetricsPath: types.StringValue("/metrics"),
				Name:        types.StringValue("Name"),
			},
			&saml2Model{},
			&basicAuthModel{},
			[]targetModel{
				{
					URLs:   types.ListValueMust(types.StringType, []attr.Value{types.StringValue("url1")}),
					Labels: types.MapValueMust(types.StringType, map[string]attr.Value{"k1": types.StringValue("v1")}),
				},
				{
					URLs:   types.ListValueMust(types.StringType, []attr.Value{types.StringValue("url1"), types.StringValue("url3")}),
					Labels: types.MapValueMust(types.StringType, map[string]attr.Value{"k2": types.StringValue("v2"), "k3": types.StringValue("v3")}),
				},
				{
					URLs:   types.ListValueMust(types.StringType, []attr.Value{}),
					Labels: types.MapNull(types.StringType),
				},
			},
			&observabilitySdk.UpdateScrapeConfigPayload{
				MetricsPath: "/metrics",
				StaticConfigs: []observabilitySdk.UpdateScrapeConfigPayloadStaticConfigsInner{
					{
						Targets: []string{"url1"},
						Labels:  map[string]any{"k1": "v1"},
					},
					{
						Targets: []string{"url1", "url3"},
						Labels:  map[string]any{"k2": "v2", "k3": "v3"},
					},
					{
						Targets: []string{},
						Labels:  map[string]any{},
					},
				},
				// Defaults
				Scheme:         "http",
				ScrapeInterval: "5m",
				ScrapeTimeout:  "2m",
				SampleLimit:    new(float32(5000)),
			},
			true,
		},
		{
			"nil_model",
			nil,
			nil,
			nil,
			nil,
			nil,
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			output, err := toUpdatePayload(context.Background(), tt.input, tt.inputSAML2, tt.basicAuthModel, tt.inputTargets)
			if !tt.isValid && err == nil {
				t.Fatalf("Should have failed")
			}
			if tt.isValid && err != nil {
				t.Fatalf("Should not have failed: %v", err)
			}
			if tt.isValid {
				diff := cmp.Diff(output, tt.expected)
				if diff != "" {
					t.Fatalf("Data does not match: %s", diff)
				}
			}
		})
	}
}
