package shared

import (
	"context"
	"fmt"
	"maps"
	"strings"

	"dev.azure.com/schwarzit/schwarzit.stackit-public/stackit-sdk-go-internal.git/services/iaasalpha"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"
)

type DataSourceModel struct {
	Id               types.String `tfsdk:"id"` // needed by TF
	OrganizationId   types.String `tfsdk:"organization_id"`
	RoutingTableId   types.String `tfsdk:"routing_table_id"`
	Name             types.String `tfsdk:"name"`
	NetworkAreaId    types.String `tfsdk:"network_area_id"`
	Description      types.String `tfsdk:"description"`
	Labels           types.Map    `tfsdk:"labels"`
	Region           types.String `tfsdk:"region"`
	MainRoutingTable types.Bool   `tfsdk:"main_routing_table"`
	SystemRoutes     types.Bool   `tfsdk:"system_routes"`
	CreatedAt        types.String `tfsdk:"created_at"`
	UpdatedAt        types.String `tfsdk:"updated_at"`
	Routes           types.List   `tfsdk:"routes"`
}

// This is needed for listing of routing tables therefore the terraform id, organization id and network area id is not needed
var DataSourceTypes = map[string]attr.Type{
	"routing_table_id":   types.StringType,
	"name":               types.StringType,
	"description":        types.StringType,
	"labels":             types.MapType{ElemType: types.StringType},
	"region":             types.StringType,
	"main_routing_table": types.BoolType,
	"system_routes":      types.BoolType,
	"created_at":         types.StringType,
	"updated_at":         types.StringType,
	// TODO: adjust
	"routes": types.ListType{ElemType: types.StringType},
}

func GetDatasourceGetAttributes() map[string]schema.Attribute {
	// combine the schemas
	getAttributes := RoutingTableResponseAttributes
	maps.Copy(getAttributes, datasourceGetAttributes)
	return getAttributes
}

var datasourceGetAttributes = map[string]schema.Attribute{
	"id": schema.StringAttribute{
		Description: "Terraform's internal datasource ID. It is structured as \"`organization_id`,`region`,`routing_table_id`\".",
		Computed:    true,
	},
	"organization_id": schema.StringAttribute{
		Description: "STACKIT organization ID to which the routing table is associated.",
		Required:    true,
		Validators: []validator.String{
			validate.UUID(),
			validate.NoSeparator(),
		},
	},
	"routing_table_id": schema.StringAttribute{
		Description: "The routing tables ID.",
		Required:    true,
		Validators: []validator.String{
			validate.UUID(),
			validate.NoSeparator(),
		},
	},
	"network_area_id": schema.StringAttribute{
		Description: "The network area ID to which the routing table is associated.",
		Required:    true,
		Validators: []validator.String{
			validate.UUID(),
			validate.NoSeparator(),
		},
	},
}

var RoutingTableResponseAttributes = map[string]schema.Attribute{
	"routing_table_id": schema.StringAttribute{
		Description: "The routing tables ID.",
		Computed:    true,
	},
	"name": schema.StringAttribute{
		Description: "The name of the routing table.",
		Computed:    true,
	},
	"description": schema.StringAttribute{
		Description: "Description of the routing table.",
		Computed:    true,
	},
	"labels": schema.MapAttribute{
		Description: "Labels are key-value string pairs which can be attached to a resource container",
		ElementType: types.StringType,
		Computed:    true,
	},
	"main_routing_table": schema.BoolAttribute{
		Description: "Sets the routing table as main routing table.",
		Computed:    true,
	},
	"system_routes": schema.BoolAttribute{
		Description: "TODO: ask what this does",
		Computed:    true,
	},
	"created_at": schema.StringAttribute{
		Description: "Date-time when the routing table was created",
		Computed:    true,
	},
	"updated_at": schema.StringAttribute{
		Description: "Date-time when the routing table was updated",
		Computed:    true,
	},
	"routes": schema.ListNestedAttribute{
		Description: "List of routes.",
		Computed:    true,
		NestedObject: schema.NestedAttributeObject{
			Attributes: map[string]schema.Attribute{
				"id": schema.StringAttribute{
					Description: "Route ID.",
					Computed:    true,
				},
				"destination": schema.SingleNestedAttribute{
					Description: "Destination of the route.",
					Computed:    true,
					Attributes: map[string]schema.Attribute{
						"type": schema.StringAttribute{
							Description: "CIDRV type.",
							Computed:    true,
						},
						"value": schema.StringAttribute{
							Description: "An CIDR string.",
							Computed:    true,
						},
					},
				},
				"next_hop": schema.SingleNestedAttribute{
					Description: "Next hop destination.",
					Computed:    true,
					Attributes: map[string]schema.Attribute{
						"type": schema.StringAttribute{
							Description: "Can be either blackhole, internet, ipv4 or ipv6.",
							Computed:    true,
						},
						"value": schema.StringAttribute{
							Description: "Either IPv4 or IPv6 (not set for blackhole and internet).",
							Computed:    true,
						},
					},
				},
				"labels": schema.MapAttribute{
					Description: "Labels are key-value string pairs which can be attached to a resource container",
					ElementType: types.StringType,
					Computed:    true,
				},
				"created_at": schema.StringAttribute{
					Description: "Date-time when the route was created",
					Computed:    true,
				},
				"updated_at": schema.StringAttribute{
					Description: "Date-time when the route was updated",
					Computed:    true,
				},
			},
		},
	},
}

// TODO: add the mapping of the Routes here (Route Model needed)
func MapDataSourceFields(ctx context.Context, routingTable *iaasalpha.RoutingTable, model *DataSourceModel, region string) error {
	if routingTable == nil {
		return fmt.Errorf("response input is nil")
	}
	if model == nil {
		return fmt.Errorf("model input is nil")
	}

	var routingTableId string
	if model.RoutingTableId.ValueString() != "" {
		routingTableId = model.RoutingTableId.ValueString()
	} else if routingTable.Id != nil {
		routingTableId = *routingTable.Id
	} else {
		return fmt.Errorf("routing table id not present")
	}

	idParts := []string{
		model.OrganizationId.ValueString(),
		region,
		routingTableId,
	}
	model.Id = types.StringValue(
		strings.Join(idParts, core.Separator),
	)

	// TODO: add mapping of routes
	/*err := mapRoutes(routingTable, model)
	if err != nil {
		return fmt.Errorf("mapping routes: %w", err)
	}*/

	labels, diags := types.MapValueFrom(ctx, types.StringType, map[string]interface{}{})
	if diags.HasError() {
		return fmt.Errorf("converting labels to StringValue map: %w", core.DiagsToError(diags))
	}

	if routingTable.Labels != nil && len(*routingTable.Labels) != 0 {
		var diags diag.Diagnostics
		labels, diags = types.MapValueFrom(ctx, types.StringType, *routingTable.Labels)
		if diags.HasError() {
			return fmt.Errorf("converting labels to StringValue map: %w", core.DiagsToError(diags))
		}
	} else if model.Labels.IsNull() {
		labels = types.MapNull(types.StringType)
	}

	model.RoutingTableId = types.StringValue(routingTableId)
	model.Name = types.StringPointerValue(routingTable.Name)
	model.Description = types.StringPointerValue(routingTable.Description)
	model.MainRoutingTable = types.BoolPointerValue(routingTable.MainRoutingTable)
	model.SystemRoutes = types.BoolPointerValue(routingTable.SystemRoutes)
	model.Labels = labels
	model.Region = types.StringValue(region)
	return nil
}
