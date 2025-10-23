package networkarearegion

import (
	"context"
	"reflect"
	"testing"

	"github.com/stackitcloud/stackit-sdk-go/services/iaas"
)

func Test_mapFields(t *testing.T) {
	type args struct {
		networkAreaRegion *iaas.RegionalArea
		model             *Model
		region            string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			if err := mapFields(ctx, tt.args.networkAreaRegion, tt.args.model, tt.args.region); (err != nil) != tt.wantErr {
				t.Errorf("mapFields() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_toCreatePayload(t *testing.T) {
	type args struct {
		ctx   context.Context
		model *Model
	}
	tests := []struct {
		name    string
		args    args
		want    iaas.CreateNetworkAreaRegionPayload
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := toCreatePayload(tt.args.ctx, tt.args.model)
			if (err != nil) != tt.wantErr {
				t.Errorf("toCreatePayload() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("toCreatePayload() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_toUpdatePayload(t *testing.T) {
	type args struct {
		ctx   context.Context
		model *Model
	}
	tests := []struct {
		name    string
		args    args
		want    iaas.UpdateNetworkAreaRegionPayload
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := toUpdatePayload(tt.args.ctx, tt.args.model)
			if (err != nil) != tt.wantErr {
				t.Errorf("toUpdatePayload() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("toUpdatePayload() got = %v, want %v", got, tt.want)
			}
		})
	}
}
