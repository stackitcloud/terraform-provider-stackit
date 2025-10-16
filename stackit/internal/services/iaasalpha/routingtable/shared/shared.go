package shared

import (
	"context"
	"fmt"
	"maps"
	"time"

	iaasUtils "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/iaas/utils"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stackitcloud/stackit-sdk-go/services/iaasalpha"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"
)

type RoutingTableReadModel struct {
	RoutingTableId types.String `tfsdk:"routing_table_id"`
	Name           types.String `tfsdk:"name"`
	Description    types.String `tfsdk:"description"`
	Labels         types.Map    `tfsdk:"labels"`
	CreatedAt      types.String `tfsdk:"created_at"`
	UpdatedAt      types.String `tfsdk:"updated_at"`
	Default        types.Bool   `tfsdk:"default"`
	SystemRoutes   types.Bool   `tfsdk:"system_routes"`
}

func RoutingTableReadModelTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"routing_table_id": types.StringType,
		"name":             types.StringType,
		"description":      types.StringType,
		"labels":           types.MapType{ElemType: types.StringType},
		"created_at":       types.StringType,
		"updated_at":       types.StringType,
		"default":          types.BoolType,
		"system_routes":    types.BoolType,
	}
}

type RoutingTableDataSourceModel struct {
	RoutingTableReadModel
	Id             types.String `tfsdk:"id"` // needed by TF
	OrganizationId types.String `tfsdk:"organization_id"`
	NetworkAreaId  types.String `tfsdk:"network_area_id"`
	Region         types.String `tfsdk:"region"`
}

func GetDatasourceGetAttributes() map[string]schema.Attribute {
	// combine the schemas
	getAttributes := RoutingTableResponseAttributes()
	maps.Copy(getAttributes, datasourceGetAttributes())
	getAttributes["id"] = schema.StringAttribute{
		Description: "Terraform's internal datasource ID. It is structured as \"`organization_id`,`region`,`network_area_id`,`routing_table_id`\".",
		Computed:    true,
	}
	return getAttributes
}

func GetRouteDataSourceAttributes() map[string]schema.Attribute {
	getAttributes := datasourceGetAttributes()
	maps.Copy(getAttributes, RouteResponseAttributes())
	getAttributes["route_id"] = schema.StringAttribute{
		Description: "Route ID.",
		Required:    true,
		Validators: []validator.String{
			validate.UUID(),
			validate.NoSeparator(),
		},
	}
	getAttributes["id"] = schema.StringAttribute{
		Description: "Terraform's internal datasource ID. It is structured as \"`organization_id`,`region`,`network_area_id`,`routing_table_id`,`route_id`\".",
		Computed:    true,
	}
	return getAttributes
}

func GetRoutesDataSourceAttributes() map[string]schema.Attribute {
	getAttributes := datasourceGetAttributes()
	getAttributes["id"] = schema.StringAttribute{
		Description: "Terraform's internal datasource ID. It is structured as \"`organization_id`,`region`,`network_area_id`,`routing_table_id`,`route_id`\".",
		Computed:    true,
	}
	getAttributes["routes"] = schema.ListNestedAttribute{
		Description: "List of routes.",
		Computed:    true,
		NestedObject: schema.NestedAttributeObject{
			Attributes: RouteResponseAttributes(),
		},
	}
	getAttributes["region"] = schema.StringAttribute{
		Description: "The datasource region. If not defined, the provider region is used.",
		Optional:    true,
	}
	return getAttributes
}

func datasourceGetAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
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
		"region": schema.StringAttribute{
			Description: "The resource region. If not defined, the provider region is used.",
			Optional:    true,
		},
	}
}

func RouteResponseAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"route_id": schema.StringAttribute{
			Description: "Route ID.",
			Computed:    true,
		},
		"destination": schema.SingleNestedAttribute{
			Description: "Destination of the route.",
			Computed:    true,
			Attributes: map[string]schema.Attribute{
				"type": schema.StringAttribute{
					Description: fmt.Sprintf("CIDRV type. %s %s", utils.FormatPossibleValues("cidrv4", "cidrv6"), "Only `cidrv4` is supported during experimental stage."),
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
					Description: "Type of the next hop. " + utils.FormatPossibleValues("blackhole", "internet", "ipv4", "ipv6"),
					Computed:    true,
				},
				"value": schema.StringAttribute{
					Description: "Either IPv4 or IPv6 (not set for blackhole and internet). Only IPv4 supported during experimental stage.",
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
	}
}

func RoutingTableResponseAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
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
		"default": schema.BoolAttribute{
			Description: "When true this is the default routing table for this network area. It can't be deleted and is used if the user does not specify it otherwise.",
			Computed:    true,
		},
		"system_routes": schema.BoolAttribute{
			Description: "This controls whether the routes for project-to-project communication are created automatically or not.",
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
	}
}

func MapRoutingTableReadModel(ctx context.Context, routingTable *iaasalpha.RoutingTable, model *RoutingTableReadModel) error {
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

	labels, err := iaasUtils.MapLabels(ctx, routingTable.Labels, model.Labels)
	if err != nil {
		return err
	}

	// created at and updated at
	createdAtTF, updatedAtTF := types.StringNull(), types.StringNull()
	if routingTable.CreatedAt != nil {
		createdAtValue := *routingTable.CreatedAt
		createdAtTF = types.StringValue(createdAtValue.Format(time.RFC3339))
	}
	if routingTable.UpdatedAt != nil {
		updatedAtValue := *routingTable.UpdatedAt
		updatedAtTF = types.StringValue(updatedAtValue.Format(time.RFC3339))
	}

	model.RoutingTableId = types.StringValue(routingTableId)
	model.Name = types.StringPointerValue(routingTable.Name)
	model.Description = types.StringPointerValue(routingTable.Description)
	model.Default = types.BoolPointerValue(routingTable.Default)
	model.SystemRoutes = types.BoolPointerValue(routingTable.SystemRoutes)
	model.Labels = labels
	model.CreatedAt = createdAtTF
	model.UpdatedAt = updatedAtTF
	return nil
}
