package clientutils

import (
	"context"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/stackitcloud/stackit-sdk-go/services/serviceenablement/v2api"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
)

func TestDefaultClientFactory_NewServiceEnablementV2Client(t *testing.T) {
	type args struct {
		ctx          context.Context
		providerData *core.ProviderData
		diags        *diag.Diagnostics
	}
	tests := []struct {
		name string
		args args
		want v2api.DefaultAPI
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &DefaultClientFactory{}
			if got := f.NewServiceEnablementV2Client(tt.args.ctx, tt.args.providerData, tt.args.diags); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewServiceEnablementV2Client() = %v, want %v", got, tt.want)
			}
		})
	}
}
