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
	"github.com/stackitcloud/stackit-sdk-go/services/edge"
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
		Id:          utils.Ptr(id),
		DisplayName: utils.Ptr(displayName),
		PlanId:      &defaultPlanId,
		FrontendUrl: utils.Ptr(fmt.Sprintf("https://%s.example.com", id)),
		Status:      utils.Ptr(edge.InstanceStatus("ACTIVE")),
		Created:     &testTime,
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
		"instance_id":  types.StringValue(*validInstance.Id),
		"display_name": types.StringValue(*validInstance.DisplayName),
		"region":       types.StringValue(region),
		"plan_id":      types.StringValue(*validInstance.PlanId),
		"frontend_url": types.StringValue(*validInstance.FrontendUrl),
		"status":       types.StringValue(string(*validInstance.Status)),
		"created":      types.StringValue(testTime.String()),
		"description":  types.StringValue(*validInstance.Description),
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
				m["description"] = types.StringValue("")
			}),
			expectError: false,
		},
		{
			description: "error, nil display name",
			instance: fixtureInstance(func(i *edge.Instance) {
				i.DisplayName = nil
			}),
			expectError: true,
			errorMsg:    "missing a 'displayName'",
		},
		{
			description: "error, empty display name",
			instance: fixtureInstance(func(i *edge.Instance) {
				i.DisplayName = utils.Ptr("")
			}),
			expectError: true,
			errorMsg:    "missing a 'displayName'",
		},
		{
			description: "error, nil id",
			instance: fixtureInstance(func(i *edge.Instance) {
				i.Id = nil
			}),
			expectError: true,
			errorMsg:    "missing an 'id'",
		},
		{
			description: "error, nil planId",
			instance: fixtureInstance(func(i *edge.Instance) {
				i.PlanId = nil
			}),
			expectError: true,
			errorMsg:    "missing a 'planId'",
		},
		{
			description: "error, nil frontendUrl",
			instance: fixtureInstance(func(i *edge.Instance) {
				i.FrontendUrl = nil
			}),
			expectError: true,
			errorMsg:    "missing a 'frontendUrl'",
		},
		{
			description: "error, nil status",
			instance: fixtureInstance(func(i *edge.Instance) {
				i.Status = nil
			}),
			expectError: true,
			errorMsg:    "missing a 'status'",
		},
		{
			description: "error, nil created",
			instance: fixtureInstance(func(i *edge.Instance) {
				i.Created = nil
			}),
			expectError: true,
			errorMsg:    "missing a 'created' timestamp",
		},
		{
			description: "error, nil description",
			instance: fixtureInstance(func(i *edge.Instance) {
				i.Description = nil
			}),
			expectError: true,
			errorMsg:    "missing a 'description'",
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
		i.Id = utils.Ptr("first-ab75568")
		i.DisplayName = utils.Ptr("first")
	})

	instance2 := fixtureInstance(func(i *edge.Instance) {
		i.Id = utils.Ptr("second-ab75568")
		i.DisplayName = utils.Ptr("second")
	})

	instanceInvalidPlan := fixtureInstance(func(i *edge.Instance) {
		i.PlanId = nil
	})

	// Invalid: Nil Display Name
	instanceNilName := fixtureInstance(func(i *edge.Instance) {
		i.DisplayName = nil
	})

	// Invalid: Empty Display Name
	instanceEmptyName := fixtureInstance(func(i *edge.Instance) {
		i.DisplayName = utils.Ptr("")
	})

	// Invalid: Nil ID and Nil Display Name
	instanceNilIdAndName := fixtureInstance(func(i *edge.Instance) {
		i.Id = nil
		i.DisplayName = nil
	})

	// Pre-calculate expected mapped objects for the valid instances
	attrs1, _ := mapInstanceToAttrs(instance1, region)
	obj1, _ := types.ObjectValue(instanceTypes, attrs1)

	attrs2, _ := mapInstanceToAttrs(instance2, region)
	obj2, _ := types.ObjectValue(instanceTypes, attrs2)

	tests := []struct {
		description       string
		instances         edge.InstanceListGetInstancesAttributeType
		expectedList      []attr.Value
		expectedDiagCount int
	}{
		{
			description:       "empty instance list",
			instances:         &[]edge.Instance{}, // No test case for nil, since this is checked before buildInstancesList is called
			expectedList:      []attr.Value{},
			expectedDiagCount: 0,
		},
		{
			description:       "two valid instances",
			instances:         &[]edge.Instance{instance1, instance2},
			expectedList:      []attr.Value{obj1, obj2},
			expectedDiagCount: 0,
		},
		{
			description:       "one valid, one invalid (nil planId)",
			instances:         &[]edge.Instance{instance1, instanceInvalidPlan},
			expectedList:      []attr.Value{obj1},
			expectedDiagCount: 1,
		},
		{
			description:       "one valid, one invalid (nil display name)",
			instances:         &[]edge.Instance{instance1, instanceNilName},
			expectedList:      []attr.Value{obj1},
			expectedDiagCount: 1,
		},
		{
			description:       "one valid, one invalid (empty display name)",
			instances:         &[]edge.Instance{instance1, instanceEmptyName},
			expectedList:      []attr.Value{obj1},
			expectedDiagCount: 1,
		},
		{
			description:       "one valid, one invalid (nil id and nil display name)",
			instances:         &[]edge.Instance{instance1, instanceNilIdAndName},
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
