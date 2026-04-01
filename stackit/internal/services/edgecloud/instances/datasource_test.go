package instances

import (
	"context"
	"fmt"
	"maps"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"

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
		Description: new(description),
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
	}{
		{
			description: "valid instance",
			instance:    validInstance,
			expected:    validInstanceAttrs,
		},
		{
			description: "valid instance, empty description",
			instance: fixtureInstance(func(i *edge.Instance) {
				i.Description = new("")
			}),
			expected: fixtureAttrs(validInstanceAttrs, func(m map[string]attr.Value) {
				m["description"] = types.StringPointerValue(new(""))
			}),
		},
		{
			description: "nil description",
			instance: fixtureInstance(func(i *edge.Instance) {
				i.Description = nil
			}),
			expected: fixtureAttrs(validInstanceAttrs, func(m map[string]attr.Value) {
				m["description"] = types.StringPointerValue(nil)
			}),
		},
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			attrs := mapInstanceToAttrs(&tt.instance, region)

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

	// Pre-calculate expected mapped objects for the valid instances
	attrs1 := mapInstanceToAttrs(&instance1, region)
	obj1, _ := types.ObjectValue(instanceTypes, attrs1)

	attrs2 := mapInstanceToAttrs(&instance2, region)
	obj2, _ := types.ObjectValue(instanceTypes, attrs2)

	tests := []struct {
		description  string
		instances    []edge.Instance
		expectedList []attr.Value
	}{
		{
			description:  "empty instance list",
			instances:    []edge.Instance{}, // No test case for nil, since this is checked before buildInstancesList is called
			expectedList: []attr.Value{},
		},
		{
			description:  "two valid instances",
			instances:    []edge.Instance{instance1, instance2},
			expectedList: []attr.Value{obj1, obj2},
		},
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			var diags diag.Diagnostics

			resultList := buildInstancesList(ctx, tt.instances, region, &diags)

			if diags.HasError() {
				t.Errorf("Expected no errors, but got diagnostics: %v", diags)
			}

			diff := cmp.Diff(tt.expectedList, resultList, cmpopts.EquateEmpty())
			if diff != "" {
				t.Errorf("Resulting list does not match expected:\n%s", diff)
			}
		})
	}
}
