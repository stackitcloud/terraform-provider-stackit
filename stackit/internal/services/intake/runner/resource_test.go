package runner

import (
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stackitcloud/stackit-sdk-go/core/utils"
	"github.com/stackitcloud/stackit-sdk-go/services/intake"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
)

func TestMapFields(t *testing.T) {
	runnerId := uuid.New().String()
	tests := []struct {
		description string
		input       *intake.IntakeRunnerResponse
		model       *Model
		region      string
		expected    *Model
		wantErr     bool
	}{
		{
			"success",
			&intake.IntakeRunnerResponse{
				Id:                 utils.Ptr(runnerId),
				DisplayName:        utils.Ptr("name"),
				Description:        utils.Ptr("description"),
				Labels:             &map[string]string{"key": "value"},
				MaxMessageSizeKiB:  utils.Ptr(int64(1024)),
				MaxMessagesPerHour: utils.Ptr(int64(100)),
			},
			&Model{
				ProjectId: types.StringValue("pid"),
			},
			"eu01",
			&Model{
				Id:                 types.StringValue(fmt.Sprintf("pid,eu01,%s", runnerId)),
				ProjectId:          types.StringValue("pid"),
				Region:             types.StringValue("eu01"),
				RunnerId:           types.StringValue(runnerId),
				Name:               types.StringValue("name"),
				Description:        types.StringValue("description"),
				Labels:             types.MapValueMust(types.StringType, map[string]attr.Value{"key": types.StringValue("value")}),
				MaxMessageSizeKiB:  types.Int64Value(1024),
				MaxMessagesPerHour: types.Int64Value(100),
			},
			false,
		},
		{
			"nil input",
			nil,
			&Model{},
			"eu01",
			nil,
			true,
		},
		{
			"nil model",
			&intake.IntakeRunnerResponse{},
			nil,
			"eu01",
			nil,
			true,
		},
		{
			"empty response",
			&intake.IntakeRunnerResponse{
				Id:     utils.Ptr(""),
				Labels: &map[string]string{},
			},
			&Model{
				ProjectId: types.StringValue("pid"),
			},
			"eu01",
			&Model{
				Id:                 types.StringValue("pid,eu01,"),
				ProjectId:          types.StringValue("pid"),
				Region:             types.StringValue("eu01"),
				RunnerId:           types.StringNull(),
				Name:               types.StringNull(),
				Description:        types.StringNull(),
				Labels:             types.MapValueMust(types.StringType, map[string]attr.Value{}),
				MaxMessageSizeKiB:  types.Int64Null(),
				MaxMessagesPerHour: types.Int64Null(),
			},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			err := mapFields(tt.input, tt.model, tt.region)
			if (err != nil) != tt.wantErr {
				t.Errorf("mapFields error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if diff := cmp.Diff(tt.expected, tt.model); diff != "" {
					t.Errorf("mapFields mismatch (-want +got):\n%s", diff)
				}
			}
		})
	}
}

func TestToCreatePayload(t *testing.T) {
	tests := []struct {
		description string
		model       *Model
		expected    *intake.CreateIntakeRunnerPayload
		wantErr     bool
	}{
		{
			"success",
			&Model{
				Name:               types.StringValue("name"),
				Description:        types.StringValue("description"),
				Labels:             types.MapValueMust(types.StringType, map[string]attr.Value{"key": types.StringValue("value")}),
				MaxMessageSizeKiB:  types.Int64Value(1024),
				MaxMessagesPerHour: types.Int64Value(100),
			},
			&intake.CreateIntakeRunnerPayload{
				DisplayName:        utils.Ptr("name"),
				Description:        utils.Ptr("description"),
				Labels:             utils.Ptr(map[string]string{"key": "value"}),
				MaxMessageSizeKiB:  utils.Ptr(int64(1024)),
				MaxMessagesPerHour: utils.Ptr(int64(100)),
			},
			false,
		},
		{
			"nil model",
			nil,
			nil,
			true,
		},
		{
			"empty model",
			&Model{},
			&intake.CreateIntakeRunnerPayload{
				DisplayName:        nil,
				Description:        nil,
				Labels:             nil,
				MaxMessageSizeKiB:  nil,
				MaxMessagesPerHour: nil,
			},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			payload, err := toCreatePayload(tt.model)
			if (err != nil) != tt.wantErr {
				t.Errorf("toCreatePayload error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if diff := cmp.Diff(tt.expected, payload); diff != "" {
					t.Errorf("toCreatePayload mismatch (-want +got):\n%s", diff)
				}
			}
		})
	}
}

func TestToUpdatePayload(t *testing.T) {
	tests := []struct {
		description string
		model       *Model
		state       *Model
		expected    *intake.UpdateIntakeRunnerPayload
		wantErr     bool
	}{
		{
			"success",
			&Model{
				Name:               types.StringValue("name"),
				Description:        types.StringValue("description"),
				Labels:             types.MapValueMust(types.StringType, map[string]attr.Value{"key": types.StringValue("value")}),
				MaxMessageSizeKiB:  types.Int64Value(1024),
				MaxMessagesPerHour: types.Int64Value(100),
			},
			&Model{},
			&intake.UpdateIntakeRunnerPayload{
				DisplayName:        conversion.StringValueToPointer(types.StringValue("name")),
				Description:        conversion.StringValueToPointer(types.StringValue("description")),
				Labels:             utils.Ptr(map[string]string{"key": "value"}),
				MaxMessageSizeKiB:  conversion.Int64ValueToPointer(types.Int64Value(1024)),
				MaxMessagesPerHour: conversion.Int64ValueToPointer(types.Int64Value(100)),
			},
			false,
		},
		{
			"nil model",
			nil,
			&Model{},
			nil,
			true,
		},
		{
			"nil state",
			&Model{},
			nil,
			nil,
			true,
		},
		{
			"empty model",
			&Model{},
			&Model{},
			&intake.UpdateIntakeRunnerPayload{},
			false,
		},
		{
			"unknown values",
			&Model{
				Name:               types.StringUnknown(),
				Description:        types.StringUnknown(),
				Labels:             types.MapUnknown(types.StringType),
				MaxMessageSizeKiB:  types.Int64Unknown(),
				MaxMessagesPerHour: types.Int64Unknown(),
			},
			&Model{},
			&intake.UpdateIntakeRunnerPayload{},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			payload, err := toUpdatePayload(tt.model, tt.state)
			if (err != nil) != tt.wantErr {
				t.Errorf("toUpdatePayload error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if diff := cmp.Diff(tt.expected, payload); diff != "" {
					t.Errorf("toUpdatePayload mismatch (-want +got):\n%s", diff)
				}
			}
		})
	}
}
