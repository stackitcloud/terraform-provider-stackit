package cdn

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stackitcloud/stackit-sdk-go/services/cdn"
)

func TestMapFields(t *testing.T) {
	emtpyErrorsList := types.ListValueMust(types.StringType, []attr.Value{})
	expectedModel := func(mods ...func(*CustomDomainModel)) *CustomDomainModel {
		model := &CustomDomainModel{
			ID:             types.StringValue("test-project-id,test-distribution-id,https://testdomain.com"),
			DistributionId: types.StringValue("test-distribution-id"),
			ProjectId:      types.StringValue("test-project-id"),
			Status:         types.StringValue("ACTIVE"),
			Errors:         emtpyErrorsList,
		}
		for _, mod := range mods {
			mod(model)
		}
		return model
	}
	customDomainFixture := func(mods ...func(*cdn.CustomDomain)) *cdn.CustomDomain {
		distribution := &cdn.CustomDomain{
			Errors: &[]cdn.StatusError{},
			Name:   cdn.PtrString("https://testdomain.com"),
			Status: cdn.DOMAINSTATUS_ACTIVE.Ptr(),
		}
		for _, mod := range mods {
			mod(distribution)
		}
		return distribution
	}
	tests := map[string]struct {
		Input    *cdn.CustomDomain
		Expected *CustomDomainModel
		IsValid  bool
	}{
		"happy_path": {
			Expected: expectedModel(),
			Input:    customDomainFixture(),
			IsValid:  true,
		},
		"happy_path_status_error": {
			Expected: expectedModel(func(m *CustomDomainModel) {
				m.Status = types.StringValue("ERROR")
			}),
			Input: customDomainFixture(func(d *cdn.CustomDomain) {
				d.Status = cdn.DOMAINSTATUS_ERROR.Ptr()
			}),
			IsValid: true,
		},
		"sad_path_custom_domain_nil": {
			Expected: expectedModel(),
			Input:    nil,
			IsValid:  false,
		},
		"sad_path_name_missing": {
			Expected: expectedModel(),
			Input: customDomainFixture(func(d *cdn.CustomDomain) {
				d.Name = nil
			}),
			IsValid: false,
		},
	}
	for tn, tc := range tests {
		t.Run(tn, func(t *testing.T) {
			model := &CustomDomainModel{}
			model.DistributionId = tc.Expected.DistributionId
			model.ProjectId = tc.Expected.ProjectId
			err := mapCustomDomainFields(tc.Input, model)
			if err != nil && tc.IsValid {
				t.Fatalf("Error mapping fields: %v", err)
			}
			if err == nil && !tc.IsValid {
				t.Fatalf("Should have failed")
			}
			if tc.IsValid {
				diff := cmp.Diff(model, tc.Expected)
				if diff != "" {
					t.Fatalf("Create Payload not as expected: %s", diff)
				}
			}
		})
	}
}
