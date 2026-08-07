package intakes

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stackitcloud/stackit-sdk-go/core/utils"
	intake "github.com/stackitcloud/stackit-sdk-go/services/intake/v1betaapi"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
)

func TestMapFields(t *testing.T) {
	intakeId := uuid.New().String()
	runnerId := uuid.New().String()
	now := time.Now()

	tests := []struct {
		description string
		input       *intake.IntakeResponse
		model       *Model
		region      string
		expected    *Model
		wantErr     bool
	}{
		{
			"success",
			&intake.IntakeResponse{
				Id:             intakeId,
				IntakeRunnerId: runnerId,
				DisplayName:    "name",
				Description:    utils.Ptr("description"),
				Labels:         map[string]string{"key": "value"},
				Uri:            "https://intake.eu01.onstackit.cloud",
				CreateTime:     now,
				Catalog: intake.IntakeCatalog{
					Namespace:    utils.Ptr("intake_ns"),
					TableName:    utils.Ptr("intake_table"),
					Uri:          "https://catalog.dremio.eu01.onstackit.cloud",
					Warehouse:    "default",
					Partitioning: utils.Ptr(intake.PartitioningType("DAY")),
					PartitionBy:  []string{"col1", "col2"},
					Auth: &intake.CatalogAuth{
						Type: intake.CatalogAuthType("dremio"),
						Dremio: &intake.DremioAuth{ //nolint:gosec // mock test data
							TokenEndpoint: "https://dremio.eu01.onstackit.cloud/oauth/endpoint",
						},
					},
				},
			},
			&Model{
				ProjectId: types.StringValue("pid"),
				DremioPAT: types.StringValue("secret_token"),
			},
			"eu01",
			&Model{
				Id:                  types.StringValue(fmt.Sprintf("pid,eu01,%s", intakeId)),
				ProjectId:           types.StringValue("pid"),
				Region:              types.StringValue("eu01"),
				IntakeId:            types.StringValue(intakeId),
				RunnerId:            types.StringValue(runnerId),
				Name:                types.StringValue("name"),
				Description:         types.StringValue("description"),
				Labels:              types.MapValueMust(types.StringType, map[string]attr.Value{"key": types.StringValue("value")}),
				Uri:                 types.StringValue("https://intake.eu01.onstackit.cloud"),
				CreateTime:          types.StringValue(now.String()),
				DremioPAT:           types.StringValue("secret_token"),
				DremioTokenEndpoint: types.StringValue("https://dremio.eu01.onstackit.cloud/oauth/endpoint"),
				CatalogAuthType:     types.StringValue("dremio"),
				CatalogNamespace:    types.StringValue("intake_ns"),
				CatalogPartitioning: types.StringValue("DAY"),
				CatalogPartitionBy:  types.ListValueMust(types.StringType, []attr.Value{types.StringValue("col1"), types.StringValue("col2")}),
				CatalogTableName:    types.StringValue("intake_table"),
				CatalogUri:          types.StringValue("https://catalog.dremio.eu01.onstackit.cloud"),
				CatalogWarehouse:    types.StringValue("default"),
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
			&intake.IntakeResponse{},
			nil,
			"eu01",
			nil,
			true,
		},
		{
			"empty response",
			&intake.IntakeResponse{
				Id:     "",
				Labels: map[string]string{},
			},
			&Model{
				ProjectId: types.StringValue("pid"),
			},
			"eu01",
			&Model{
				Id:                  types.StringValue("pid,eu01,"),
				ProjectId:           types.StringValue("pid"),
				Region:              types.StringValue("eu01"),
				IntakeId:            types.StringValue(""),
				RunnerId:            types.StringValue(""),
				Name:                types.StringValue(""),
				Description:         types.StringNull(),
				Labels:              types.MapNull(types.StringType),
				Uri:                 types.StringValue(""),
				CreateTime:          types.StringValue(time.Time{}.String()),
				DremioPAT:           types.StringNull(),
				DremioTokenEndpoint: types.StringNull(),
				CatalogAuthType:     types.StringNull(),
				CatalogNamespace:    types.StringNull(),
				CatalogPartitioning: types.StringNull(),
				CatalogPartitionBy:  types.ListNull(types.StringType),
				CatalogTableName:    types.StringNull(),
				CatalogUri:          types.StringValue(""),
				CatalogWarehouse:    types.StringValue(""),
			},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			err := mapFields(context.Background(), tt.input, tt.model, tt.region)
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
	runnerId := uuid.New().String()

	tests := []struct {
		description string
		model       *Model
		expected    *intake.CreateIntakePayload
		wantErr     bool
	}{
		{
			"success",
			&Model{
				RunnerId:            types.StringValue(runnerId),
				Name:                types.StringValue("name"),
				Description:         types.StringValue("description"),
				Labels:              types.MapValueMust(types.StringType, map[string]attr.Value{"key": types.StringValue("value")}),
				DremioPAT:           types.StringValue("token"),
				DremioTokenEndpoint: types.StringValue("https://dremio.eu01.onstackit.cloud/oauth/endpoint"),
				CatalogAuthType:     types.StringValue("dremio"),
				CatalogNamespace:    types.StringValue("ns"),
				CatalogPartitioning: types.StringValue("intake-time"),
				CatalogPartitionBy:  types.ListValueMust(types.StringType, []attr.Value{types.StringValue("col1"), types.StringValue("col2")}),
				CatalogTableName:    types.StringValue("table"),
				CatalogUri:          types.StringValue("https://catalog.uri"),
				CatalogWarehouse:    types.StringValue("wh"),
			},
			&intake.CreateIntakePayload{
				IntakeRunnerId: runnerId,
				DisplayName:    "name",
				Description:    utils.Ptr("description"),
				Labels:         map[string]string{"key": "value"},
				Catalog: intake.IntakeCatalog{
					Auth: &intake.CatalogAuth{
						Type:   intake.CatalogAuthType("dremio"),
						Dremio: intake.NewDremioAuth("token", "https://dremio.eu01.onstackit.cloud/oauth/endpoint"),
					},
					Namespace:    utils.Ptr("ns"),
					PartitionBy:  []string{"col1", "col2"},
					Partitioning: utils.Ptr(intake.PartitioningType("intake-time")),
					TableName:    utils.Ptr("table"),
					Uri:          "https://catalog.uri",
					Warehouse:    "wh",
				},
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
			&intake.CreateIntakePayload{
				IntakeRunnerId: "",
				DisplayName:    "",
				Description:    nil,
				Labels:         map[string]string{},
				Catalog: intake.IntakeCatalog{
					Auth:         nil,
					Namespace:    nil,
					PartitionBy:  nil,
					Partitioning: nil,
					TableName:    nil,
					Uri:          "",
					Warehouse:    "",
				},
			},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			payload, err := toCreatePayload(context.Background(), tt.model)
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
		expected    *intake.UpdateIntakePayload
		wantErr     bool
	}{
		{
			"success",
			&Model{
				Name:        types.StringValue("name"),
				Description: types.StringValue("description"),
				Labels:      types.MapValueMust(types.StringType, map[string]attr.Value{"key": types.StringValue("value")}),
			},
			&intake.UpdateIntakePayload{
				DisplayName: conversion.StringValueToPointer(types.StringValue("name")),
				Description: conversion.StringValueToPointer(types.StringValue("description")),
				Labels:      map[string]string{"key": "value"},
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
			&intake.UpdateIntakePayload{
				Labels: map[string]string{},
			},
			false,
		},
		{
			"unknown values",
			&Model{
				Name:        types.StringUnknown(),
				Description: types.StringUnknown(),
				Labels:      types.MapUnknown(types.StringType),
			},
			&intake.UpdateIntakePayload{
				Labels: map[string]string{},
			},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			payload, err := toUpdatePayload(context.Background(), tt.model)
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
