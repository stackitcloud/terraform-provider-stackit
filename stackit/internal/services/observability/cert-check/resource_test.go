package certcheck

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
		certCheck           *observability.CertCheckChildResponse
		model               *Model
		expectedId          string
		expectedCertCheckId string
		expectedSource      string
		expectErr           bool
	}{
		{
			name:      "Nil CertCheck",
			certCheck: nil,
			model:     &Model{},
			expectErr: true,
		},
		{
			name:      "Nil Model",
			certCheck: &observability.CertCheckChildResponse{},
			model:     nil,
			expectErr: true,
		},
		{
			name: "Complete Model and CertCheck",
			certCheck: &observability.CertCheckChildResponse{
				Id:     utils.Ptr("cert-check-id"),
				Source: utils.Ptr("cert-check-source"),
			},
			model: &Model{
				ProjectId:  types.StringValue("project1"),
				InstanceId: types.StringValue("instance1"),
			},
			expectedId:          "project1,instance1,cert-check-id",
			expectedCertCheckId: "cert-check-id",
			expectedSource:      "cert-check-source",
			expectErr:           false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			err := mapFields(ctx, tt.certCheck, tt.model)

			if (err != nil) != tt.expectErr {
				t.Fatalf("expected error: %v, got: %v", tt.expectErr, err)
			}

			if !tt.expectErr {
				if diff := cmp.Diff(tt.model.Id.ValueString(), tt.expectedId); diff != "" {
					t.Errorf("unexpected ID (-got +want):\n%s", diff)
				}
				if diff := cmp.Diff(tt.model.CertCheckId.ValueString(), tt.expectedCertCheckId); diff != "" {
					t.Errorf("unexpected CertCheckId (-got +want):\n%s", diff)
				}
				if diff := cmp.Diff(tt.model.Source.ValueString(), tt.expectedSource); diff != "" {
					t.Errorf("unexpected Source (-got +want):\n%s", diff)
				}
			}
		})
	}
}
