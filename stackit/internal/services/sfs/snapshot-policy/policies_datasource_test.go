package snapshot_policy

import (
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework/types"
	sfs "github.com/stackitcloud/stackit-sdk-go/services/sfs/v1api"
)

func TestMapFields(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name    string
		resp    *sfs.ListSnapshotPoliciesResponse
		want    *model
		wantErr bool
	}{
		{
			name:    "nil response",
			resp:    nil,
			want:    &model{},
			wantErr: true,
		},
		{
			name: "some values",
			resp: &sfs.ListSnapshotPoliciesResponse{
				SnapshotPolicies: []sfs.SnapshotPolicy{
					{
						Comment:   new("comment"),
						CreatedAt: new(now),
						Enabled:   new(true),
						Id:        new("id"),
						Name:      new("name"),
						SnapshotSchedules: []sfs.SnapshotPolicySnapshotPolicySchedule{
							{
								CreatedAt:       new(now),
								Id:              new("id"),
								Interval:        new("interval"),
								Name:            new("name"),
								Prefix:          new("prefix"),
								RetentionCount:  new(int32(123)),
								RetentionPeriod: new("period"),
							},
						},
					},
				},
			},
			want: &model{
				ID: types.StringValue(""),
				Items: []policy{
					{
						ID:        types.StringValue("id"),
						Name:      types.StringValue("name"),
						Comment:   types.StringValue("comment"),
						Enabled:   types.BoolValue(true),
						CreatedAt: types.StringValue(now.String()),
						SnapshotSchedules: []schedule{
							{
								ID:              types.StringValue("id"),
								Name:            types.StringValue("name"),
								CreatedAt:       types.StringValue(now.String()),
								Interval:        types.StringValue("interval"),
								Prefix:          types.StringValue("prefix"),
								RetentionCount:  types.Int64Value(123),
								RetentionPeriod: types.StringValue("period"),
							},
						},
					},
				},
			},
		},
		{
			name: "nil values policy",
			resp: &sfs.ListSnapshotPoliciesResponse{
				SnapshotPolicies: []sfs.SnapshotPolicy{
					{},
				},
			},
			want: &model{
				ID: types.StringValue(""),
				Items: []policy{
					{
						ID:                types.String{},
						Name:              types.String{},
						Comment:           types.String{},
						Enabled:           types.Bool{},
						CreatedAt:         types.String{},
						SnapshotSchedules: nil,
					},
				},
			},
		},
		{
			name: "nil values schedule",
			resp: &sfs.ListSnapshotPoliciesResponse{
				SnapshotPolicies: []sfs.SnapshotPolicy{
					{
						SnapshotSchedules: []sfs.SnapshotPolicySnapshotPolicySchedule{
							{},
						},
					},
				},
			},
			want: &model{
				ID: types.StringValue(""),
				Items: []policy{
					{
						SnapshotSchedules: []schedule{
							{},
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &model{}
			err := mapFields(t.Context(), tt.resp, m)
			if err != nil && !tt.wantErr {
				t.Errorf("unexpected error: %v", err)
			}
			if err == nil && tt.wantErr {
				t.Errorf("unexpected error: %v", err)
			}
			if err != nil {
				return
			}
			if diff := cmp.Diff(tt.want, m); diff != "" {
				t.Errorf("Data does not match: %s", diff)
			}
		})
	}
}
