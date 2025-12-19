// Copyright (c) STACKIT

package core

import (
	"testing"

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
			name: "(legacy) region is set",
			providerData: &ProviderData{
				Region: "eu02",
			},
			want: "eu02",
		},
		{
			name: "default region wins over (legacy) region",
			providerData: &ProviderData{
				DefaultRegion: "eu02",
				Region:        "eu01",
			},
			want: "eu02",
		},
		{
			name:         "final fallback - neither region (legacy) nor default region is set",
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
