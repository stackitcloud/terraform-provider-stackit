package instances

import (
	"context"
	"fmt"
	"maps"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/stackitcloud/stackit-sdk-go/core/utils"
	edge "github.com/stackitcloud/stackit-sdk-go/services/edge/v1beta1api"
)

// testTime is a shared helper for generating consistent timestamps
var testTime, _ = time.Parse(time.RFC3339, "2023-09-04T10:00:00Z")

// defaultPlanId is defined outside the function to avoid it having a different value each call
var defaultPlanId = uuid.NewString()

// fixtureInstance creates a valid default instance and applies modifiers.
func fixtureInstance(mods ...func(instance *edge.Instance)) edge.Instance {
	id := "some-hash"
	displayName := "some"
	description := "some-description"

	instance := &edge.Instance{
		Id:          id,
		DisplayName: displayName,
		PlanId:      defaultPlanId,
		FrontendUrl: fmt.Sprintf("https://%s.example.com", id),
		Status:      "ACTIVE",
		Created:     testTime,
		Description: utils.Ptr(description),
	}

	for _, mod := range mods {
		mod(instance)
	}

	return *instance
}

func fixtureAttrs(base map[string]attr.Value, mods ...func(m map[string]attr.Value)) map[string]attr.Value {
	m := maps.Clone(base)
	for _, mod := range mods {
		mod(m)
	}
	return m
}

func TestMapInstanceToAttrs(t *testing.T) {
	region := "eu01"
	validInstance := fixtureInstance()

	validInstanceAttrs := map[string]attr.Value{
		"instance_id":  types.StringValue(validInstance.Id),
		"display_name": types.StringValue(validInstance.DisplayName),
		"region":       types.StringValue(region),
		"plan_id":      types.StringValue(validInstance.PlanId),
		"frontend_url": types.StringValue(validInstance.FrontendUrl),
		"status":       types.StringValue(validInstance.Status),
		"created":      types.StringValue(testTime.String()),
		"description":  types.StringPointerValue(validInstance.Description),
	}

	tests := []struct {
		description string
		instance    edge.Instance
		expected    map[string]attr.Value
		expectError bool
		errorMsg    string
	}{
		{
			description: "valid instance",
			instance:    validInstance,
			expected:    validInstanceAttrs,
			expectError: false,
		},
		{
			description: "valid instance, empty description",
			instance: fixtureInstance(func(i *edge.Instance) {
				i.Description = utils.Ptr("")
			}),
			expected: fixtureAttrs(validInstanceAttrs, func(m map[string]attr.Value) {
				m["description"] = types.StringPointerValue(utils.Ptr(""))
			}),
			expectError: false,
		},
		{
			description: "error, empty display name",
			instance: fixtureInstance(func(i *edge.Instance) {
				i.DisplayName = ""
			}),
			expectError: true,
			errorMsg:    "missing a 'displayName'",
		},
		{
			description: "error, empty id",
			instance: fixtureInstance(func(i *edge.Instance) {
				i.Id = ""
			}),
			expectError: true,
			errorMsg:    "missing an 'id'",
		},
		{
			description: "error, empty planId",
			instance: fixtureInstance(func(i *edge.Instance) {
				i.PlanId = ""
			}),
			expectError: true,
			errorMsg:    "missing a 'planId'",
		},
		{
			description: "error, empty frontendUrl",
			instance: fixtureInstance(func(i *edge.Instance) {
				i.FrontendUrl = ""
			}),
			expectError: true,
			errorMsg:    "missing a 'frontendUrl'",
		},
		{
			description: "error, empty status",
			instance: fixtureInstance(func(i *edge.Instance) {
				i.Status = ""
			}),
			expectError: true,
			errorMsg:    "missing a 'status'",
		},
		{
			description: "error, nil description",
			instance: fixtureInstance(func(i *edge.Instance) {
				i.Description = nil
			}),
			expectError: false,
			expected: fixtureAttrs(validInstanceAttrs, func(m map[string]attr.Value) {
				m["description"] = types.StringPointerValue(nil)
			}),
		},
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			attrs, err := mapInstanceToAttrs(tt.instance, region)

			if tt.expectError {
				if err == nil {
					t.Fatalf("Expected an error, but got nil")
				}
				if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error message to contain %q, but got: %v", tt.errorMsg, err)
				}
				return
			}

			if err != nil {
				t.Fatalf("Expected no error, but got: %v", err)
			}

			diff := cmp.Diff(tt.expected, attrs, cmpopts.EquateEmpty())
			if diff != "" {
				t.Errorf("Resulting attributes do not match expected:\n%s", diff)
			}
		})
	}
}

func TestBuildInstancesList(t *testing.T) {
	region := "eu01"
	ctx := context.Background()

	instance1 := fixtureInstance(func(i *edge.Instance) {
		i.Id = "first-ab75568"
		i.DisplayName = "first"
	})

	instance2 := fixtureInstance(func(i *edge.Instance) {
		i.Id = "second-ab75568"
		i.DisplayName = "second"
	})

	instanceInvalidPlan := fixtureInstance(func(i *edge.Instance) {
		i.PlanId = ""
	})

	// Invalid: Empty Display Name
	instanceEmptyName := fixtureInstance(func(i *edge.Instance) {
		i.DisplayName = ""
	})

	// Invalid: Empty ID and Empty Display Name
	instanceEmptyIdAndName := fixtureInstance(func(i *edge.Instance) {
		i.Id = ""
		i.DisplayName = ""
	})

	// Pre-calculate expected mapped objects for the valid instances
	attrs1, _ := mapInstanceToAttrs(instance1, region)
	obj1, _ := types.ObjectValue(instanceTypes, attrs1)

	attrs2, _ := mapInstanceToAttrs(instance2, region)
	obj2, _ := types.ObjectValue(instanceTypes, attrs2)

	tests := []struct {
		description       string
		instances         []edge.Instance
		expectedList      []attr.Value
		expectedDiagCount int
	}{
		{
			description:       "empty instance list",
			instances:         []edge.Instance{}, // No test case for nil, since this is checked before buildInstancesList is called
			expectedList:      []attr.Value{},
			expectedDiagCount: 0,
		},
		{
			description:       "two valid instances",
			instances:         []edge.Instance{instance1, instance2},
			expectedList:      []attr.Value{obj1, obj2},
			expectedDiagCount: 0,
		},
		{
			description:       "one valid, one invalid (empty planId)",
			instances:         []edge.Instance{instance1, instanceInvalidPlan},
			expectedList:      []attr.Value{obj1},
			expectedDiagCount: 1,
		},
		{
			description:       "one valid, one invalid (empty display name)",
			instances:         []edge.Instance{instance1, instanceEmptyName},
			expectedList:      []attr.Value{obj1},
			expectedDiagCount: 1,
		},
		{
			description:       "one valid, one invalid (empty id and empty display name)",
			instances:         []edge.Instance{instance1, instanceEmptyIdAndName},
			expectedList:      []attr.Value{obj1},
			expectedDiagCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			var diags diag.Diagnostics

			resultList := buildInstancesList(ctx, tt.instances, region, &diags)

			if tt.expectedDiagCount > 0 {
				if !diags.HasError() {
					t.Errorf("Expected diagnostics to have errors, but it didn't")
				}
				if len(diags) != tt.expectedDiagCount {
					t.Errorf("Expected %d diagnostic(s), but got %d", tt.expectedDiagCount, len(diags))
				}
				for _, d := range diags {
					if d.Severity() != diag.SeverityError {
						t.Errorf("Expected diagnostic to be an Error, but got %v", d.Severity())
					}
				}
			} else if diags.HasError() {
				t.Errorf("Expected no errors, but got diagnostics: %v", diags)
			}

			diff := cmp.Diff(tt.expectedList, resultList, cmpopts.EquateEmpty())
			if diff != "" {
				t.Errorf("Resulting list does not match expected:\n%s", diff)
			}
		})
	}
}
