package postgresflexalpha

import (
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
)

// type postgresFlexClientMocked struct {
//	returnError    bool
//	getFlavorsResp *postgresflex.GetFlavorsResponse
// }
//
// func (c *postgresFlexClientMocked) ListFlavorsExecute(_ context.Context, _, _ string) (*postgresflex.GetFlavorsResponse, error) {
//	if c.returnError {
//		return nil, fmt.Errorf("get flavors failed")
//	}
//
//	return c.getFlavorsResp, nil
// }

func TestNewInstanceResource(t *testing.T) {
	tests := []struct {
		name string
		want resource.Resource
	}{
		{
			name: "create empty instance resource",
			want: &instanceResource{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewInstanceResource(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewInstanceResource() = %v, want %v", got, tt.want)
			}
		})
	}
}
