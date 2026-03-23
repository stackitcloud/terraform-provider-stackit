package utils

import (
	"context"
	"os"
	"reflect"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	sdkClients "github.com/stackitcloud/stackit-sdk-go/core/clients"
	"github.com/stackitcloud/stackit-sdk-go/core/config"
	utils2 "github.com/stackitcloud/stackit-sdk-go/core/utils"
	"github.com/stackitcloud/stackit-sdk-go/services/sfs"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"
)

const (
	testVersion        = "1.2.3"
	testCustomEndpoint = "https://sfs-custom-endpoint.api.stackit.cloud"
)

func TestConfigureClient(t *testing.T) {
	/* mock authentication by setting service account token env variable */
	os.Clearenv()
	err := os.Setenv(sdkClients.ServiceAccountToken, "mock-val")
	if err != nil {
		t.Errorf("error setting env variable: %v", err)
	}

	type args struct {
		providerData *core.ProviderData
	}
	tests := []struct {
		name     string
		args     args
		wantErr  bool
		expected *sfs.APIClient
	}{
		{
			name: "default endpoint",
			args: args{
				providerData: &core.ProviderData{
					Version: testVersion,
				},
			},
			expected: func() *sfs.APIClient {
				apiClient, err := sfs.NewAPIClient(
					utils.UserAgentConfigOption(testVersion),
				)
				if err != nil {
					t.Errorf("error configuring client: %v", err)
				}
				return apiClient
			}(),
			wantErr: false,
		},
		{
			name: "custom endpoint",
			args: args{
				providerData: &core.ProviderData{
					Version:           testVersion,
					SfsCustomEndpoint: testCustomEndpoint,
				},
			},
			expected: func() *sfs.APIClient {
				apiClient, err := sfs.NewAPIClient(
					utils.UserAgentConfigOption(testVersion),
					config.WithEndpoint(testCustomEndpoint),
				)
				if err != nil {
					t.Errorf("error configuring client: %v", err)
				}
				return apiClient
			}(),
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			diags := diag.Diagnostics{}

			actual := ConfigureClient(ctx, tt.args.providerData, &diags)
			if diags.HasError() != tt.wantErr {
				t.Errorf("ConfigureClient() error = %v, want %v", diags.HasError(), tt.wantErr)
			}

			if !reflect.DeepEqual(actual, tt.expected) {
				t.Errorf("ConfigureClient() = %v, want %v", actual, tt.expected)
			}
		})
	}
}

func TestDescribeValidationError(t *testing.T) {
	tests := []struct {
		name string
		err  sfs.ValidationError
		want string
	}{
		{
			name: "just title",
			err: sfs.ValidationError{
				Title: utils2.Ptr("nice title"),
			},
			want: `nice title
`,
		},
		{
			name: "with fields",
			err: sfs.ValidationError{
				Title: utils2.Ptr("nice title"),
				Fields: &[]sfs.ValidationErrorField{
					{
						Field:  utils2.Ptr("field-a"),
						Reason: utils2.Ptr("reason-a"),
					},
					{
						Reason: utils2.Ptr("reason-b"),
					},
					{
						Field: utils2.Ptr("field-c"),
					},
				},
			},
			want: `nice title

Field: field-a | Reason: reason-a
Field:  | Reason: reason-b
Field: field-c | Reason: `,
		},
	}

	for _, tt := range tests {
		got := DescribeValidationError(tt.err)
		if d := cmp.Diff(got, tt.want); d != "" {
			t.Errorf("DescribeValidationError() = got diff: %s", d)
		}
	}
}
