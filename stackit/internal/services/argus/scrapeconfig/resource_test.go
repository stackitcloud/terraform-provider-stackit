package argus

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stackitcloud/stackit-sdk-go/core/utils"
	"github.com/stackitcloud/stackit-sdk-go/services/argus"
)

func TestMapFields(t *testing.T) {
	tests := []struct {
		description string
		input       *argus.Job
		expected    Model
		isValid     bool
	}{
		{
			"default_ok",
			&argus.Job{
				JobName: utils.Ptr("name"),
			},
			Model{
				Id:             types.StringValue("pid,iid,name"),
				ProjectId:      types.StringValue("pid"),
				InstanceId:     types.StringValue("iid"),
				Name:           types.StringValue("name"),
				MetricsPath:    types.StringNull(),
				Scheme:         types.StringNull(),
				ScrapeInterval: types.StringNull(),
				ScrapeTimeout:  types.StringNull(),
				SAML2:          types.ObjectNull(saml2Types),
				BasicAuth:      nil,
				Targets:        []targetModel{},
			},
			true,
		},
		{
			description: "values_ok",
			input: &argus.Job{
				JobName:     utils.Ptr("name"),
				MetricsPath: utils.Ptr("/m"),
				BasicAuth: &argus.BasicAuth{
					Password: utils.Ptr("p"),
					Username: utils.Ptr("u"),
				},
				Params:         &map[string][]string{"saml2": {"disabled"}, "x": {"y", "z"}},
				Scheme:         utils.Ptr("scheme"),
				ScrapeInterval: utils.Ptr("1"),
				ScrapeTimeout:  utils.Ptr("2"),
				SampleLimit:    utils.Ptr(int64(17)),
				StaticConfigs: &[]argus.StaticConfigs{
					{
						Labels:  &map[string]string{"k1": "v1"},
						Targets: &[]string{"url1"},
					},
					{
						Labels:  &map[string]string{"k2": "v2", "k3": "v3"},
						Targets: &[]string{"url1", "url3"},
					},
					{
						Labels:  nil,
						Targets: &[]string{},
					},
				},
			},
			expected: Model{
				Id:             types.StringValue("pid,iid,name"),
				ProjectId:      types.StringValue("pid"),
				InstanceId:     types.StringValue("iid"),
				Name:           types.StringValue("name"),
				MetricsPath:    types.StringValue("/m"),
				Scheme:         types.StringValue("scheme"),
				ScrapeInterval: types.StringValue("1"),
				ScrapeTimeout:  types.StringValue("2"),
				SampleLimit:    types.Int64Value(17),
				SAML2: types.ObjectValueMust(saml2Types, map[string]attr.Value{
					"enable_url_parameters": types.BoolValue(false),
				}),
				BasicAuth: &basicAuthModel{
					Username: types.StringValue("u"),
					Password: types.StringValue("p"),
				},
				Targets: []targetModel{
					{
						URLs: []types.String{types.StringValue("url1")},
						Labels: types.MapValueMust(types.StringType, map[string]attr.Value{
							"k1": types.StringValue("v1"),
						}),
					},
					{
						URLs: []types.String{types.StringValue("url1"), types.StringValue("url3")},
						Labels: types.MapValueMust(types.StringType, map[string]attr.Value{
							"k2": types.StringValue("v2"),
							"k3": types.StringValue("v3"),
						}),
					},
					{
						URLs:   []types.String{},
						Labels: types.MapNull(types.StringType),
					},
				},
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
			&argus.Job{},
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
			err := mapFields(tt.input, state)
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
		description string
		input       *Model
		inputSAML2  *saml2Model
		expected    *argus.CreateScrapeConfigPayload
		isValid     bool
	}{
		{
			"basic_ok",
			&Model{
				MetricsPath: types.StringValue("/metrics"),
			},
			&saml2Model{},
			&argus.CreateScrapeConfigPayload{
				MetricsPath: utils.Ptr("/metrics"),
				// Defaults
				Scheme:         utils.Ptr("https"),
				ScrapeInterval: utils.Ptr("5m"),
				ScrapeTimeout:  utils.Ptr("2m"),
				SampleLimit:    utils.Ptr(float64(5000)),
				StaticConfigs:  &[]argus.CreateScrapeConfigPayloadStaticConfigsInner{},
				Params:         &map[string]any{"saml2": []string{"enabled"}},
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
			&argus.CreateScrapeConfigPayload{
				MetricsPath: utils.Ptr("/metrics"),
				JobName:     utils.Ptr("Name"),
				// Defaults
				Scheme:         utils.Ptr("https"),
				ScrapeInterval: utils.Ptr("5m"),
				ScrapeTimeout:  utils.Ptr("2m"),
				SampleLimit:    utils.Ptr(float64(5000)),
				StaticConfigs:  &[]argus.CreateScrapeConfigPayloadStaticConfigsInner{},
				Params:         &map[string]any{"saml2": []string{"disabled"}},
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
			&argus.CreateScrapeConfigPayload{
				MetricsPath: utils.Ptr("/metrics"),
				JobName:     utils.Ptr("Name"),
				// Defaults
				Scheme:         utils.Ptr("https"),
				ScrapeInterval: utils.Ptr("5m"),
				ScrapeTimeout:  utils.Ptr("2m"),
				SampleLimit:    utils.Ptr(float64(5000)),
				StaticConfigs:  &[]argus.CreateScrapeConfigPayloadStaticConfigsInner{},
				Params:         &map[string]any{"saml2": []string{"enabled"}},
			},
			true,
		},
		{
			"nil_model",
			nil,
			nil,
			nil,
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			output, err := toCreatePayload(context.Background(), tt.input, tt.inputSAML2)
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
		description string
		input       *Model
		inputSAML2  *saml2Model
		expected    *argus.UpdateScrapeConfigPayload
		isValid     bool
	}{
		{
			"basic_ok",
			&Model{
				MetricsPath: types.StringValue("/metrics"),
			},
			&saml2Model{},
			&argus.UpdateScrapeConfigPayload{
				MetricsPath: utils.Ptr("/metrics"),
				// Defaults
				Scheme:         utils.Ptr("https"),
				ScrapeInterval: utils.Ptr("5m"),
				ScrapeTimeout:  utils.Ptr("2m"),
				SampleLimit:    utils.Ptr(float64(5000)),
				StaticConfigs:  &[]argus.UpdateScrapeConfigPayloadStaticConfigsInner{},
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
			&argus.UpdateScrapeConfigPayload{
				MetricsPath: utils.Ptr("/metrics"),
				// Defaults
				Scheme:         utils.Ptr("http"),
				ScrapeInterval: utils.Ptr("5m"),
				ScrapeTimeout:  utils.Ptr("2m"),
				SampleLimit:    utils.Ptr(float64(5000)),
				StaticConfigs:  &[]argus.UpdateScrapeConfigPayloadStaticConfigsInner{},
				Params:         &map[string]any{"saml2": []string{"enabled"}},
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
			&argus.UpdateScrapeConfigPayload{
				MetricsPath: utils.Ptr("/metrics"),
				// Defaults
				Scheme:         utils.Ptr("http"),
				ScrapeInterval: utils.Ptr("5m"),
				ScrapeTimeout:  utils.Ptr("2m"),
				SampleLimit:    utils.Ptr(float64(5000)),
				StaticConfigs:  &[]argus.UpdateScrapeConfigPayloadStaticConfigsInner{},
				Params:         &map[string]any{"saml2": []string{"disabled"}},
			},
			true,
		},
		{
			"nil_model",
			nil,
			nil,
			nil,
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			output, err := toUpdatePayload(context.Background(), tt.input, tt.inputSAML2)
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
