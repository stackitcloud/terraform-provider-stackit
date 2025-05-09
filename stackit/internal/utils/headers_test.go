package utils

import (
	"reflect"
	"testing"

	"github.com/stackitcloud/stackit-sdk-go/core/config"
)

func TestUserAgentConfigOption(t *testing.T) {
	type args struct {
		providerVersion string
	}
	tests := []struct {
		name string
		args args
		want config.ConfigurationOption
	}{
		{
			name: "TestUserAgentConfigOption",
			args: args{
				providerVersion: "1.0.0",
			},
			want: config.WithUserAgent("stackit-terraform-provider/1.0.0"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clientConfigActual := config.Configuration{}
			err := tt.want(&clientConfigActual)
			if err != nil {
				t.Errorf("error applying configuration: %v", err)
			}

			clientConfigExpected := config.Configuration{}
			err = UserAgentConfigOption(tt.args.providerVersion)(&clientConfigExpected)
			if err != nil {
				t.Errorf("error applying configuration: %v", err)
			}

			if !reflect.DeepEqual(clientConfigActual, clientConfigExpected) {
				t.Errorf("UserAgentConfigOption() = %v, want %v", clientConfigActual, clientConfigExpected)
			}
		})
	}
}
