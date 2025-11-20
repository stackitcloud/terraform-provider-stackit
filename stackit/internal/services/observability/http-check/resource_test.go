package httpcheck

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stackitcloud/stackit-sdk-go/core/utils"
	"github.com/stackitcloud/stackit-sdk-go/services/observability"
)

func TestMapFields(t *testing.T) {
	tests := []struct {
		name                string
		httpCheck           *observability.HttpCheckChildResponse
		model               *Model
		expectedId          string
		expectedHttpCheckId string
		expectedUrl         string
		expectErr           bool
	}{
		{
			name:      "Nil HttpCheck",
			httpCheck: nil,
			model:     &Model{},
			expectErr: true,
		},
		{
			name:      "Nil Model",
			httpCheck: &observability.HttpCheckChildResponse{},
			model:     nil,
			expectErr: true,
		},
		{
			name: "Complete Model and HttpCheck",
			httpCheck: &observability.HttpCheckChildResponse{
				Id:  utils.Ptr("http-check-id"),
				Url: utils.Ptr("https://example.com"),
			},
			model: &Model{
				ProjectId:  types.StringValue("project1"),
				InstanceId: types.StringValue("instance1"),
			},
			expectedId:          "project1,instance1,http-check-id",
			expectedHttpCheckId: "http-check-id",
			expectedUrl:         "https://example.com",
			expectErr:           false,
		},
		{
			name: "Nil HttpCheck Id",
			httpCheck: &observability.HttpCheckChildResponse{
				Url: utils.Ptr("https://example.com"),
			},
			model: &Model{
				ProjectId:  types.StringValue("project1"),
				InstanceId: types.StringValue("instance1"),
			},
			expectErr: true,
		},
		{
			name: "Nil HttpCheck Url",
			httpCheck: &observability.HttpCheckChildResponse{
				Id: utils.Ptr("http-check-id"),
			},
			model: &Model{
				ProjectId:  types.StringValue("project1"),
				InstanceId: types.StringValue("instance1"),
			},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			err := mapFields(ctx, tt.httpCheck, tt.model)

			if (err != nil) != tt.expectErr {
				t.Fatalf("expected error: %v, got: %v", tt.expectErr, err)
			}

			if !tt.expectErr {
				if diff := cmp.Diff(tt.model.Id.ValueString(), tt.expectedId); diff != "" {
					t.Errorf("unexpected ID (-got +want):\n%s", diff)
				}
				if diff := cmp.Diff(tt.model.HttpCheckId.ValueString(), tt.expectedHttpCheckId); diff != "" {
					t.Errorf("unexpected HttpCheckId (-got +want):\n%s", diff)
				}
				if diff := cmp.Diff(tt.model.Url.ValueString(), tt.expectedUrl); diff != "" {
					t.Errorf("unexpected Url (-got +want):\n%s", diff)
				}
			}
		})
	}
}
