package utils

import (
	"context"
	"net/http"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/stackitcloud/stackit-sdk-go/core/config"
	"github.com/stackitcloud/stackit-sdk-go/services/iaas/v2api/wait"
)

func TestMapLabels(t *testing.T) {
	type args struct {
		responseLabels map[string]any
		currentLabels  types.Map
	}
	tests := []struct {
		name    string
		args    args
		want    basetypes.MapValue
		wantErr bool
	}{
		{
			name: "response labels is set",
			args: args{
				responseLabels: map[string]any{
					"foo1": "bar1",
					"foo2": "bar2",
				},
				currentLabels: types.MapUnknown(types.StringType),
			},
			wantErr: false,
			want: types.MapValueMust(types.StringType, map[string]attr.Value{
				"foo1": types.StringValue("bar1"),
				"foo2": types.StringValue("bar2"),
			}),
		},
		{
			name: "response labels is set but empty",
			args: args{
				responseLabels: map[string]any{},
				currentLabels:  types.MapUnknown(types.StringType),
			},
			wantErr: false,
			want:    types.MapValueMust(types.StringType, map[string]attr.Value{}),
		},
		{
			name: "response labels is nil and model labels is nil",
			args: args{
				responseLabels: nil,
				currentLabels:  types.MapNull(types.StringType),
			},
			wantErr: false,
			want:    types.MapNull(types.StringType),
		},
		{
			name: "response labels is nil and model labels is set",
			args: args{
				responseLabels: nil,
				currentLabels: types.MapValueMust(types.StringType, map[string]attr.Value{
					"foo1": types.StringValue("bar1"),
					"foo2": types.StringValue("bar2"),
				}),
			},
			wantErr: false,
			want:    types.MapValueMust(types.StringType, map[string]attr.Value{}),
		},
		{
			name: "response labels is nil and model labels is set but empty",
			args: args{
				responseLabels: nil,
				currentLabels:  types.MapValueMust(types.StringType, map[string]attr.Value{}),
			},
			wantErr: false,
			want:    types.MapValueMust(types.StringType, map[string]attr.Value{}),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			got, err := MapLabels(ctx, tt.args.responseLabels, tt.args.currentLabels)
			if (err != nil) != tt.wantErr {
				t.Errorf("MapLabels() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("MapLabels() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestReadXRequestId(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name    string
		ctx     context.Context
		want    string
		wantErr bool
	}{
		{
			name: "success",
			ctx: context.WithValue(ctx, config.ContextHTTPResponse,
				new(new(http.Response{
					Header: http.Header{
						wait.XRequestIDHeader: []string{"x-request-id"},
					},
				})),
			),
			want:    "x-request-id",
			wantErr: false,
		},
		{
			name: "empty header",
			ctx: context.WithValue(ctx, config.ContextHTTPResponse,
				new(new(http.Response{
					Header: http.Header{},
				})),
			),
			wantErr: true,
		},
		{
			name: "empty x-request-id header",
			ctx: context.WithValue(ctx, config.ContextHTTPResponse,
				new(new(http.Response{
					Header: http.Header{
						wait.XRequestIDHeader: []string{},
					},
				})),
			),
			wantErr: true,
		},
		{
			name: "invalid type (simple pointer)",
			ctx: context.WithValue(ctx, config.ContextHTTPResponse,
				new(http.Response{
					Header: http.Header{
						wait.XRequestIDHeader: []string{"x-request-id"},
					},
				}),
			),
			wantErr: true,
		},
		{
			name:    "http response with wrong type",
			ctx:     context.WithValue(ctx, config.ContextHTTPResponse, "response"),
			wantErr: true,
		},
		{
			name:    "no http response",
			ctx:     ctx,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ReadXRequestId(tt.ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("ReadXRequestId() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ReadXRequestId() got = %v, want %v", got, tt.want)
			}
		})
	}
}
