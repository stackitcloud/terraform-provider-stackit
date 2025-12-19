package instance

import (
	"context"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stackitcloud/stackit-sdk-go/core/utils"
	"github.com/stackitcloud/stackit-sdk-go/services/logs"
)

var testTime = time.Now()

func fixtureInstance(mods ...func(instance *logs.LogsInstance)) *logs.LogsInstance {
	instance := &logs.LogsInstance{
		Id:      utils.Ptr("iid"),
		Created: utils.Ptr(testTime),
		Status:  utils.Ptr(logs.LOGSINSTANCESTATUS_ACTIVE),
	}
	for _, mod := range mods {
		mod(instance)
	}
	return instance
}

func fixtureModel(mods ...func(model *Model)) *Model {
	model := &Model{
		ID:            types.StringValue("pid,rid,iid"),
		InstanceID:    types.StringValue("iid"),
		Region:        types.StringValue("rid"),
		ProjectID:     types.StringValue("pid"),
		ACL:           types.ListNull(types.StringType),
		Created:       types.StringValue(testTime.String()),
		DatasourceURL: types.String{},
		Description:   types.String{},
		DisplayName:   types.String{},
		IngestOTLPURL: types.String{},
		IngestURL:     types.String{},
		QueryRangeURL: types.String{},
		QueryURL:      types.String{},
		RetentionDays: types.Int64{},
		Status:        types.StringValue(string(logs.LOGSINSTANCESTATUS_ACTIVE)),
	}
	for _, mod := range mods {
		mod(model)
	}
	return model
}

func TestMapFields(t *testing.T) {
	tests := []struct {
		description string
		input       *logs.LogsInstance
		expected    *Model
		wantErr     bool
	}{
		{
			description: "min values",
			input:       fixtureInstance(),
			expected:    fixtureModel(),
		},
		{
			description: "max values",
			input: fixtureInstance(func(instance *logs.LogsInstance) {
				instance.Acl = &[]string{"acl-entry-1", "acl-entry-2"}
				instance.DatasourceUrl = utils.Ptr("datasource-url")
				instance.Description = utils.Ptr("description")
				instance.DisplayName = utils.Ptr("display-name")
				instance.IngestOtlpUrl = utils.Ptr("ingest-otlp-url")
				instance.IngestUrl = utils.Ptr("ingest-url")
				instance.QueryRangeUrl = utils.Ptr("query-range-url")
				instance.QueryUrl = utils.Ptr("query-url")
				instance.RetentionDays = utils.Ptr(int64(7))
			}),
			expected: fixtureModel(func(model *Model) {
				model.ACL = types.ListValueMust(types.StringType, []attr.Value{
					types.StringValue("acl-entry-1"),
					types.StringValue("acl-entry-2"),
				})
				model.DatasourceURL = types.StringValue("datasource-url")
				model.Description = types.StringValue("description")
				model.DisplayName = types.StringValue("display-name")
				model.IngestOTLPURL = types.StringValue("ingest-otlp-url")
				model.IngestURL = types.StringValue("ingest-url")
				model.QueryRangeURL = types.StringValue("query-range-url")
				model.QueryURL = types.StringValue("query-url")
				model.RetentionDays = types.Int64Value(7)
			}),
		},
		{
			description: "nil input",
			wantErr:     true,
			expected:    fixtureModel(),
		},
		{
			description: "nil status",
			input: fixtureInstance(func(instance *logs.LogsInstance) {
				instance.Status = nil
			}),
			expected: fixtureModel(),
			wantErr:  true,
		},
		{
			description: "nil created",
			input: fixtureInstance(func(instance *logs.LogsInstance) {
				instance.Created = nil
			}),
			expected: fixtureModel(),
			wantErr:  true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			state := &Model{
				ProjectID: tt.expected.ProjectID,
				Region:    tt.expected.Region,
			}
			err := mapFields(context.Background(), tt.input, state)
			if tt.wantErr && err == nil {
				t.Fatalf("Should have failed")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("Should not have failed: %v", err)
			}
			if !tt.wantErr {
				diff := cmp.Diff(state, tt.expected)
				if diff != "" {
					t.Fatalf("Data does not match: %s", diff)
				}
			}
		})
	}
}

func TestToCreatePayload(t *testing.T) {
	tests := []struct {
		description    string
		model          *Model
		expected       *logs.CreateLogsInstancePayload
		wantErrMessage string
	}{
		{
			description: "min values",
			model:       fixtureModel(),
			expected:    &logs.CreateLogsInstancePayload{},
		},
		{
			description: "max values",
			model: fixtureModel(func(model *Model) {
				model.ACL = types.ListValueMust(types.StringType, []attr.Value{
					types.StringValue("acl-entry-1"),
					types.StringValue("acl-entry-2"),
				})
				model.Description = types.StringValue("description")
				model.DisplayName = types.StringValue("display-name")
				model.RetentionDays = types.Int64Value(7)
			}),
			expected: &logs.CreateLogsInstancePayload{
				Acl:           &[]string{"acl-entry-1", "acl-entry-2"},
				Description:   utils.Ptr("description"),
				DisplayName:   utils.Ptr("display-name"),
				RetentionDays: utils.Ptr(int64(7)),
			},
		},
		{
			description:    "nil model",
			wantErrMessage: "missing model",
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			got, err := toCreatePayload(tt.model)
			if tt.wantErrMessage != "" && (err == nil || err.Error() != tt.wantErrMessage) {
				t.Fatalf("Expected error: %v, got: %v", tt.wantErrMessage, err)
			}
			if tt.wantErrMessage == "" && err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			diff := cmp.Diff(got, tt.expected)
			if diff != "" {
				t.Fatalf("Payload does not match: %s", diff)
			}
		})
	}
}

func TestToUpdatePayload(t *testing.T) {
	tests := []struct {
		description    string
		model          *Model
		expected       *logs.UpdateLogsInstancePayload
		wantErrMessage string
	}{
		{
			description: "min values",
			model:       fixtureModel(),
			expected:    &logs.UpdateLogsInstancePayload{},
		},
		{
			description: "max values",
			model: fixtureModel(func(model *Model) {
				model.ACL = types.ListValueMust(types.StringType, []attr.Value{
					types.StringValue("acl-entry-1"),
					types.StringValue("acl-entry-2"),
				})
				model.Description = types.StringValue("description")
				model.DisplayName = types.StringValue("display-name")
				model.RetentionDays = types.Int64Value(7)
			}),
			expected: &logs.UpdateLogsInstancePayload{
				Acl:           &[]string{"acl-entry-1", "acl-entry-2"},
				Description:   utils.Ptr("description"),
				DisplayName:   utils.Ptr("display-name"),
				RetentionDays: utils.Ptr(int64(7)),
			},
		},
		{
			description:    "nil model",
			wantErrMessage: "missing model",
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			got, err := toUpdatePayload(tt.model)
			if tt.wantErrMessage != "" && (err == nil || err.Error() != tt.wantErrMessage) {
				t.Fatalf("Expected error: %v, got: %v", tt.wantErrMessage, err)
			}
			if tt.wantErrMessage == "" && err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			diff := cmp.Diff(got, tt.expected)
			if diff != "" {
				t.Fatalf("Payload does not match: %s", diff)
			}
		})
	}
}
