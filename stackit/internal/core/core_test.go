package core

import (
	"context"
	"crypto/tls"
	"net/http"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestProviderData_GetRegionWithOverride(t *testing.T) {
	type args struct {
		overrideRegion types.String
	}
	tests := []struct {
		name         string
		providerData *ProviderData
		args         args
		want         string
	}{
		{
			name: "override region is null string",
			providerData: &ProviderData{
				DefaultRegion: "eu02",
			},
			args: args{
				types.StringNull(),
			},
			want: "eu02",
		},
		{
			name: "override region is unknown string",
			providerData: &ProviderData{
				DefaultRegion: "eu02",
			},
			args: args{
				types.StringUnknown(),
			},
			want: "eu02",
		},
		{
			name: "override region is set",
			providerData: &ProviderData{
				DefaultRegion: "eu02",
			},
			args: args{
				types.StringValue("eu01"),
			},
			want: "eu01",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.providerData.GetRegionWithOverride(tt.args.overrideRegion); got != tt.want {
				t.Errorf("GetRegionWithOverride() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestProviderData_GetRegion(t *testing.T) {
	tests := []struct {
		name         string
		providerData *ProviderData
		want         string
	}{
		{
			name: "default region is set",
			providerData: &ProviderData{
				DefaultRegion: "eu02",
			},
			want: "eu02",
		},
		{
			name:         "no default region, use fallback",
			providerData: &ProviderData{},
			want:         "eu01",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.providerData.GetRegion(); got != tt.want {
				t.Errorf("GetRegion() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseProviderData(t *testing.T) {
	type args struct {
		providerData any
	}

	type want struct {
		ok               bool
		providerData     ProviderData
		clientCollection ClientCollection
	}

	tests := []struct {
		name    string
		args    args
		want    want
		wantErr bool
	}{
		{
			name: "provider has not been configured",
			args: args{
				providerData: nil,
			},
			want: want{
				ok: false,
			},
			wantErr: false,
		},
		{
			name: "invalid provider data",
			args: args{
				providerData: struct{}{},
			},
			want: want{
				ok: false,
			},
			wantErr: true,
		},
		{
			name: "valid provider data",
			args: args{
				providerData: providerDataInternal{
					providerData: ProviderData{},
					clients:      ClientCollection{},
				},
			},
			want: want{
				ok:               true,
				providerData:     ProviderData{},
				clientCollection: ClientCollection{},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			diags := diag.Diagnostics{}

			actual, clients, ok := ParseProviderData(ctx, tt.args.providerData, &diags)
			if diags.HasError() != tt.wantErr {
				t.Errorf("ConfigureClient() error = %v, want %v", diags.HasError(), tt.wantErr)
			}

			if ok != tt.want.ok {
				t.Errorf("ParseProviderData() got = %v, want %v", ok, tt.want.ok)
			}

			if !reflect.DeepEqual(actual, tt.want.providerData) {
				t.Errorf("ParseProviderData() ProviderData got = %v, want %v", actual, tt.want)
			}

			if !reflect.DeepEqual(clients, tt.want.clientCollection) {
				t.Errorf("ParseProviderData() ClientCollection got = %v, want %v", actual, tt.want)
			}
		})
	}
}

func TestParseEphemeralProviderData(t *testing.T) {
	var randomRoundTripper http.RoundTripper = &http.Transport{
		TLSClientConfig: &tls.Config{MinVersion: tls.VersionTLS13},
	}

	type args struct {
		providerData any
	}

	type want struct {
		ok               bool
		providerData     EphemeralProviderData
		clientCollection ClientCollection
	}

	tests := []struct {
		name    string
		args    args
		want    want
		wantErr bool
	}{
		{
			name: "provider has not been configured",
			args: args{
				providerData: nil,
			},
			want: want{
				ok: false,
			},
			wantErr: false,
		},
		{
			name: "invalid provider data",
			args: args{
				providerData: struct{}{},
			},
			want: want{
				ok: false,
			},
			wantErr: true,
		},
		{
			name: "valid provider data 1",
			args: args{
				providerData: ephemeralProviderDataInternal{},
			},
			want: want{
				ok:               true,
				providerData:     EphemeralProviderData{},
				clientCollection: ClientCollection{},
			},
			wantErr: false,
		},
		{
			name: "valid provider data 2",
			args: args{
				providerData: ephemeralProviderDataInternal{
					ephemeralProviderData: EphemeralProviderData{
						ProviderData: ProviderData{},
						RoundTripper: randomRoundTripper,
					},
				},
			},
			want: want{
				ok: true,
				providerData: EphemeralProviderData{
					ProviderData: ProviderData{},
					RoundTripper: randomRoundTripper,
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			diags := diag.Diagnostics{}

			actual, clients, ok := ParseEphemeralProviderData(ctx, tt.args.providerData, &diags)
			if diags.HasError() != tt.wantErr {
				t.Errorf("ConfigureClient() error = %v, want %v", diags.HasError(), tt.wantErr)
			}

			if ok != tt.want.ok {
				t.Errorf("ParseEphemeralProviderData() got = %v, want %v", ok, tt.want.ok)
			}

			if !reflect.DeepEqual(actual, tt.want.providerData) {
				t.Errorf("ParseEphemeralProviderData() EphemeralProviderData got = %v, want %v", actual, tt.want)
			}

			if !reflect.DeepEqual(clients, tt.want.clientCollection) {
				t.Errorf("ParseEphemeralProviderData() ClientCollection got = %v, want %v", actual, tt.want)
			}
		})
	}
}
