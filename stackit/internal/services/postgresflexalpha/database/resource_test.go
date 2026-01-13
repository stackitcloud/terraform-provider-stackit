package postgresflexalpha

import (
	"reflect"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework/types"
	postgresflex "github.com/mhenselin/terraform-provider-stackitprivatepreview/pkg/postgresflexalpha"
	"github.com/stackitcloud/stackit-sdk-go/core/utils"
)

func TestMapFields(t *testing.T) {
	const testRegion = "region"
	tests := []struct {
		description string
		input       *postgresflex.ListDatabase
		region      string
		expected    Model
		isValid     bool
	}{
		{
			"default_values",
			&postgresflex.ListDatabase{
				Id: utils.Ptr(int64(1)),
			},
			testRegion,
			Model{
				Id:         types.StringValue("pid,region,iid,1"),
				DatabaseId: types.Int64Value(int64(1)),
				InstanceId: types.StringValue("iid"),
				ProjectId:  types.StringValue("pid"),
				Name:       types.StringNull(),
				Owner:      types.StringNull(),
				Region:     types.StringValue(testRegion),
			},
			true,
		},
		{
			"simple_values",
			&postgresflex.ListDatabase{
				Id:    utils.Ptr(int64(1)),
				Name:  utils.Ptr("dbname"),
				Owner: utils.Ptr("username"),
			},
			testRegion,
			Model{
				Id:         types.StringValue("pid,region,iid,1"),
				DatabaseId: types.Int64Value(int64(1)),
				InstanceId: types.StringValue("iid"),
				ProjectId:  types.StringValue("pid"),
				Name:       types.StringValue("dbname"),
				Owner:      types.StringValue("username"),
				Region:     types.StringValue(testRegion),
			},
			true,
		},
		{
			"null_fields_and_int_conversions",
			&postgresflex.ListDatabase{
				Id:    utils.Ptr(int64(1)),
				Name:  utils.Ptr(""),
				Owner: utils.Ptr(""),
			},
			testRegion,
			Model{
				Id:         types.StringValue("pid,region,iid,1"),
				DatabaseId: types.Int64Value(int64(1)),
				InstanceId: types.StringValue("iid"),
				ProjectId:  types.StringValue("pid"),
				Name:       types.StringValue(""),
				Owner:      types.StringValue(""),
				Region:     types.StringValue(testRegion),
			},
			true,
		},
		{
			"nil_response",
			nil,
			testRegion,
			Model{},
			false,
		},
		{
			"empty_response",
			&postgresflex.ListDatabase{},
			testRegion,
			Model{},
			false,
		},
		{
			"no_resource_id",
			&postgresflex.ListDatabase{
				Id:    utils.Ptr(int64(0)),
				Name:  utils.Ptr("dbname"),
				Owner: utils.Ptr("username"),
			},
			testRegion,
			Model{},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(
			tt.description, func(t *testing.T) {
				state := &Model{
					ProjectId:  tt.expected.ProjectId,
					InstanceId: tt.expected.InstanceId,
				}
				err := mapFields(tt.input, state, tt.region)
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
			},
		)
	}
}

func TestToCreatePayload(t *testing.T) {
	tests := []struct {
		description string
		input       *Model
		expected    *postgresflex.CreateDatabaseRequestPayload
		isValid     bool
	}{
		{
			"default_values",
			&Model{
				Name:  types.StringValue("dbname"),
				Owner: types.StringValue("username"),
			},
			&postgresflex.CreateDatabaseRequestPayload{
				Name:  utils.Ptr("dbname"),
				Owner: utils.Ptr("username"),
			},
			true,
		},
		{
			"null_fields",
			&Model{
				Name:  types.StringNull(),
				Owner: types.StringNull(),
			},
			&postgresflex.CreateDatabaseRequestPayload{
				Name:  nil,
				Owner: nil,
			},
			true,
		},
		{
			"nil_model",
			nil,
			nil,
			false,
		},
	}
	for _, tt := range tests {
		t.Run(
			tt.description, func(t *testing.T) {
				output, err := toCreatePayload(tt.input)
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
			},
		)
	}
}

func Test_cleanString(t *testing.T) {
	type args struct {
		s *string
	}
	tests := []struct {
		name string
		args args
		want *string
	}{
		{
			name: "simple_value",
			args: args{
				s: utils.Ptr("mytest"),
			},
			want: utils.Ptr("mytest"),
		},
		{
			name: "simple_value_with_quotes",
			args: args{
				s: utils.Ptr("\"mytest\""),
			},
			want: utils.Ptr("mytest"),
		},
		{
			name: "simple_values_with_quotes",
			args: args{
				s: utils.Ptr("\"my test here\""),
			},
			want: utils.Ptr("my test here"),
		},
		{
			name: "simple_values",
			args: args{
				s: utils.Ptr("my test here"),
			},
			want: utils.Ptr("my test here"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := cleanString(tt.args.s); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("cleanString() = %v, want %v", got, tt.want)
			}
		})
	}
}
