package templates

import (
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework/types"
	automation "github.com/stackitcloud/stackit-sdk-go/services/automation/v1betaapi"
)

var testTimestampValue = "2006-01-02T15:04:05Z"

func testTimestamp() time.Time {
	timestamp, _ := time.Parse(time.RFC3339, testTimestampValue)
	return timestamp
}

func TestMapFields(t *testing.T) {
	tests := []struct {
		name              string
		recievedTemplates []automation.Template
		model             *model
		expected          *model
		valid             bool
	}{
		{
			name: "maps and sorts templates",
			recievedTemplates: []automation.Template{
				{
					Id:                   "template-2",
					Description:          "second",
					Name:                 "second template",
					CreateTime:           testTimestamp(),
					AdditionalProperties: map[string]interface{}{"test": "key"},
				},
				{
					Id:                   "template-1",
					Description:          "first",
					Name:                 "first template",
					CreateTime:           testTimestamp(),
					AdditionalProperties: map[string]interface{}{"test": "key"},
				},
			},
			model: &model{
				ProjectId: types.StringValue("project-id"),
				Region:    types.StringValue("eu01"),
			},
			expected: &model{
				ID:        types.StringValue("project-id,eu01"),
				ProjectId: types.StringValue("project-id"),
				Region:    types.StringValue("eu01"),
				Templates: []template{
					{
						TemplateId:  types.StringValue("template-1"),
						Description: types.StringValue("first"),
						CreateTime:  types.StringValue(testTimestampValue),
						Name:        types.StringValue("first template"),
					},
					{
						TemplateId:  types.StringValue("template-2"),
						Description: types.StringValue("second"),
						CreateTime:  types.StringValue(testTimestampValue),
						Name:        types.StringValue("second template"),
					},
				},
			},
			valid: true,
		},
		{
			name:              "maps empty response",
			recievedTemplates: []automation.Template{},
			model: &model{
				ProjectId: types.StringValue("project-id"),
				Region:    types.StringValue("eu01"),
			},
			expected: &model{
				ID:        types.StringValue("project-id,eu01"),
				ProjectId: types.StringValue("project-id"),
				Region:    types.StringValue("eu01"),
				Templates: []template{},
			},
			valid: true,
		},
		{
			name: "maps nil response",
			model: &model{
				ProjectId: types.StringValue("project-id"),
				Region:    types.StringValue("eu01"),
			},
			expected: &model{
				ID:        types.StringValue("project-id,eu01"),
				ProjectId: types.StringValue("project-id"),
				Region:    types.StringValue("eu01"),
				Templates: nil,
			},
			valid: true,
		},
		{
			name:              "rejects nil model",
			recievedTemplates: []automation.Template{},
			valid:             false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := mapFields(tt.recievedTemplates, tt.model)
			if (err == nil) != tt.valid {
				t.Fatalf("mapFields() error = %v, valid = %t", err, tt.valid)
			}
			if tt.valid {
				if diff := cmp.Diff(tt.expected, tt.model); diff != "" {
					t.Fatalf("mapFields() mismatch (-want +got):\n%s", diff)
				}
			}
		})
	}
}
