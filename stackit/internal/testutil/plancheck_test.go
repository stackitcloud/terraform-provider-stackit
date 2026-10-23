package testutil

import (
	"context"
	"strings"
	"testing"

	tfjson "github.com/hashicorp/terraform-json"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
)

func TestExpectOnlyEphemeralOpen(t *testing.T) {
	tests := []struct {
		name            string
		configAddresses []string
		plan            tfjson.Plan
		wantErr         string
	}{
		{
			name:            "empty plan",
			configAddresses: []string{"stackit_ske_kubeconfig.test"},
			plan:            tfjson.Plan{},
		},
		{
			name:            "resource with nil change is ignored",
			configAddresses: []string{"stackit_ske_kubeconfig.test"},
			plan: tfjson.Plan{
				ResourceChanges: []*tfjson.ResourceChange{
					{
						Address: "stackit_ske_cluster.test",
						Mode:    tfjson.ManagedResourceMode,
						Change:  nil,
					},
				},
			},
		},
		{
			name:            "resource with no-op actions is ignored",
			configAddresses: []string{"stackit_ske_kubeconfig.test"},
			plan: tfjson.Plan{
				ResourceChanges: []*tfjson.ResourceChange{
					{
						Address: "stackit_ske_cluster.test",
						Mode:    tfjson.ManagedResourceMode,
						Change: &tfjson.Change{
							Actions: tfjson.Actions{tfjson.ActionNoop},
						},
					},
				},
			},
		},
		{
			name:            "ephemeral resource with single open action matching configured address succeeds",
			configAddresses: []string{"stackit_ske_kubeconfig.test"},
			plan: tfjson.Plan{
				ResourceChanges: []*tfjson.ResourceChange{
					{
						Address: "stackit_ske_kubeconfig.test",
						Mode:    tfjson.ResourceMode("ephemeral"),
						Change: &tfjson.Change{
							Actions: tfjson.Actions{tfjson.Action("open")},
						},
					},
				},
			},
		},
		{
			name:            "multiple configured ephemeral resources with open action succeed",
			configAddresses: []string{"stackit_ske_kubeconfig.test1", "stackit_ske_kubeconfig.test2"},
			plan: tfjson.Plan{
				ResourceChanges: []*tfjson.ResourceChange{
					{
						Address: "stackit_ske_kubeconfig.test1",
						Mode:    tfjson.ResourceMode("ephemeral"),
						Change: &tfjson.Change{
							Actions: tfjson.Actions{tfjson.Action("open")},
						},
					},
					{
						Address: "stackit_ske_kubeconfig.test2",
						Mode:    tfjson.ResourceMode("ephemeral"),
						Change: &tfjson.Change{
							Actions: tfjson.Actions{tfjson.Action("open")},
						},
					},
				},
			},
		},
		{
			name:            "ephemeral open resource not in configAddresses fails",
			configAddresses: []string{"stackit_ske_kubeconfig.other"},
			plan: tfjson.Plan{
				ResourceChanges: []*tfjson.ResourceChange{
					{
						Address: "stackit_ske_kubeconfig.test",
						Mode:    tfjson.ResourceMode("ephemeral"),
						Change: &tfjson.Change{
							Actions: tfjson.Actions{tfjson.Action("open")},
						},
					},
				},
			},
			wantErr: "unexpected planned action(s) [open] for stackit_ske_kubeconfig.test",
		},
		{
			name:            "managed resource matching address and open action fails due to mode",
			configAddresses: []string{"stackit_ske_kubeconfig.test"},
			plan: tfjson.Plan{
				ResourceChanges: []*tfjson.ResourceChange{
					{
						Address: "stackit_ske_kubeconfig.test",
						Mode:    tfjson.ManagedResourceMode,
						Change: &tfjson.Change{
							Actions: tfjson.Actions{tfjson.Action("open")},
						},
					},
				},
			},
			wantErr: "unexpected planned action(s) [open] for stackit_ske_kubeconfig.test",
		},
		{
			name:            "data resource matching address and open action fails due to mode",
			configAddresses: []string{"stackit_ske_kubeconfig.test"},
			plan: tfjson.Plan{
				ResourceChanges: []*tfjson.ResourceChange{
					{
						Address: "stackit_ske_kubeconfig.test",
						Mode:    tfjson.DataResourceMode,
						Change: &tfjson.Change{
							Actions: tfjson.Actions{tfjson.Action("open")},
						},
					},
				},
			},
			wantErr: "unexpected planned action(s) [open] for stackit_ske_kubeconfig.test",
		},
		{
			name:            "ephemeral resource matching address but with create action fails",
			configAddresses: []string{"stackit_ske_kubeconfig.test"},
			plan: tfjson.Plan{
				ResourceChanges: []*tfjson.ResourceChange{
					{
						Address: "stackit_ske_kubeconfig.test",
						Mode:    tfjson.ResourceMode("ephemeral"),
						Change: &tfjson.Change{
							Actions: tfjson.Actions{tfjson.ActionCreate},
						},
					},
				},
			},
			wantErr: "unexpected planned action(s) [create] for stackit_ske_kubeconfig.test",
		},
		{
			name:            "ephemeral resource matching address but with multiple actions fails",
			configAddresses: []string{"stackit_ske_kubeconfig.test"},
			plan: tfjson.Plan{
				ResourceChanges: []*tfjson.ResourceChange{
					{
						Address: "stackit_ske_kubeconfig.test",
						Mode:    tfjson.ResourceMode("ephemeral"),
						Change: &tfjson.Change{
							Actions: tfjson.Actions{tfjson.Action("open"), tfjson.ActionUpdate},
						},
					},
				},
			},
			wantErr: "unexpected planned action(s) [open update] for stackit_ske_kubeconfig.test",
		},
		{
			name:            "ephemeral resource matching address but with empty actions fails",
			configAddresses: []string{"stackit_ske_kubeconfig.test"},
			plan: tfjson.Plan{
				ResourceChanges: []*tfjson.ResourceChange{
					{
						Address: "stackit_ske_kubeconfig.test",
						Mode:    tfjson.ResourceMode("ephemeral"),
						Change: &tfjson.Change{
							Actions: tfjson.Actions{},
						},
					},
				},
			},
			wantErr: "unexpected planned action(s) [] for stackit_ske_kubeconfig.test",
		},
		{
			name:            "managed resource with create action fails",
			configAddresses: []string{"stackit_ske_kubeconfig.test"},
			plan: tfjson.Plan{
				ResourceChanges: []*tfjson.ResourceChange{
					{
						Address: "stackit_ske_cluster.test",
						Mode:    tfjson.ManagedResourceMode,
						Change: &tfjson.Change{
							Actions: tfjson.Actions{tfjson.ActionCreate},
						},
					},
				},
			},
			wantErr: "unexpected planned action(s) [create] for stackit_ske_cluster.test",
		},
		{
			name:            "mixed resource changes: nil, no-op, and allowed ephemeral open succeed",
			configAddresses: []string{"stackit_ske_kubeconfig.test"},
			plan: tfjson.Plan{
				ResourceChanges: []*tfjson.ResourceChange{
					{
						Address: "stackit_ske_cluster.nil_change",
						Mode:    tfjson.ManagedResourceMode,
						Change:  nil,
					},
					{
						Address: "stackit_ske_cluster.noop_change",
						Mode:    tfjson.ManagedResourceMode,
						Change: &tfjson.Change{
							Actions: tfjson.Actions{tfjson.ActionNoop},
						},
					},
					{
						Address: "stackit_ske_kubeconfig.test",
						Mode:    tfjson.ResourceMode("ephemeral"),
						Change: &tfjson.Change{
							Actions: tfjson.Actions{tfjson.Action("open")},
						},
					},
				},
			},
		},
		{
			name:            "nil output change is ignored",
			configAddresses: []string{"stackit_ske_kubeconfig.test"},
			plan: tfjson.Plan{
				OutputChanges: map[string]*tfjson.Change{
					"test_output": nil,
				},
			},
		},
		{
			name:            "no-op output change is ignored",
			configAddresses: []string{"stackit_ske_kubeconfig.test"},
			plan: tfjson.Plan{
				OutputChanges: map[string]*tfjson.Change{
					"test_output": {
						Actions: tfjson.Actions{tfjson.ActionNoop},
					},
				},
			},
		},
		{
			name:            "output change with create action fails",
			configAddresses: []string{"stackit_ske_kubeconfig.test"},
			plan: tfjson.Plan{
				OutputChanges: map[string]*tfjson.Change{
					"kubeconfig": {
						Actions: tfjson.Actions{tfjson.ActionCreate},
					},
				},
			},
			wantErr: "unexpected planned action(s) [create] for output kubeconfig",
		},
		{
			name:            "valid ephemeral open but output change with create action fails",
			configAddresses: []string{"stackit_ske_kubeconfig.test"},
			plan: tfjson.Plan{
				ResourceChanges: []*tfjson.ResourceChange{
					{
						Address: "stackit_ske_kubeconfig.test",
						Mode:    tfjson.ResourceMode("ephemeral"),
						Change: &tfjson.Change{
							Actions: tfjson.Actions{tfjson.Action("open")},
						},
					},
				},
				OutputChanges: map[string]*tfjson.Change{
					"kubeconfig": {
						Actions: tfjson.Actions{tfjson.ActionCreate},
					},
				},
			},
			wantErr: "unexpected planned action(s) [create] for output kubeconfig",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			check := ExpectOnlyEphemeralOpen(tt.configAddresses...)
			req := plancheck.CheckPlanRequest{
				Plan: &tt.plan,
			}
			resp := &plancheck.CheckPlanResponse{}

			check.CheckPlan(context.Background(), req, resp)

			if tt.wantErr == "" {
				if resp.Error != nil {
					t.Fatalf("unexpected error: %v", resp.Error)
				}
			} else {
				if resp.Error == nil {
					t.Fatalf("expected error containing %q, got nil", tt.wantErr)
				}
				if !strings.Contains(resp.Error.Error(), tt.wantErr) {
					t.Errorf("error %q does not contain %q", resp.Error.Error(), tt.wantErr)
				}
			}
		})
	}
}
