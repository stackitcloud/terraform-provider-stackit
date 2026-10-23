package objectstorage

import (
	_ "embed"
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework/types"
	objectstorage "github.com/stackitcloud/stackit-sdk-go/services/objectstorage/v2api"
)

func TestMapFields(t *testing.T) {
	const testRegion = "eu01"
	id := fmt.Sprintf("%s,%s,%s", "pid", testRegion, "bname")
	tests := []struct {
		description string
		input       *objectstorage.GetBucketResponse
		expected    Model
		isValid     bool
	}{
		{
			"default_values",
			&objectstorage.GetBucketResponse{
				Bucket: objectstorage.Bucket{},
			},
			Model{
				Id:                    types.StringValue(id),
				Name:                  types.StringValue("bname"),
				ProjectId:             types.StringValue("pid"),
				URLPathStyle:          types.StringValue(""),
				URLVirtualHostedStyle: types.StringValue(""),
				Region:                types.StringValue("eu01"),
				ObjectLock:            types.BoolValue(false),
			},
			true,
		},
		{
			"simple_values",
			&objectstorage.GetBucketResponse{
				Bucket: objectstorage.Bucket{
					UrlPathStyle:          "url/path/style",
					UrlVirtualHostedStyle: "url/virtual/hosted/style",
					ObjectLockEnabled:     true,
				},
			},
			Model{
				Id:                    types.StringValue(id),
				Name:                  types.StringValue("bname"),
				ProjectId:             types.StringValue("pid"),
				ObjectLock:            types.BoolValue(true),
				URLPathStyle:          types.StringValue("url/path/style"),
				URLVirtualHostedStyle: types.StringValue("url/virtual/hosted/style"),
				Region:                types.StringValue("eu01"),
			},
			true,
		},
		{
			"empty_strings",
			&objectstorage.GetBucketResponse{
				Bucket: objectstorage.Bucket{
					UrlPathStyle:          "",
					UrlVirtualHostedStyle: "",
				},
			},
			Model{
				Id:                    types.StringValue(id),
				Name:                  types.StringValue("bname"),
				ProjectId:             types.StringValue("pid"),
				URLPathStyle:          types.StringValue(""),
				URLVirtualHostedStyle: types.StringValue(""),
				Region:                types.StringValue("eu01"),
				ObjectLock:            types.BoolValue(false),
			},
			true,
		},
		{
			"nil_response",
			nil,
			Model{},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			model := &Model{
				ProjectId: tt.expected.ProjectId,
				Name:      tt.expected.Name,
			}
			err := mapFields(tt.input, model, "eu01")
			if !tt.isValid && err == nil {
				t.Fatalf("Should have failed")
			}
			if tt.isValid && err != nil {
				t.Fatalf("Should not have failed: %v", err)
			}
			if tt.isValid {
				diff := cmp.Diff(model, &tt.expected)
				if diff != "" {
					t.Fatalf("Data does not match: %s", diff)
				}
			}
		})
	}
}
