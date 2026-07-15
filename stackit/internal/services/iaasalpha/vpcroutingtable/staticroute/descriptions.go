package staticroute

import (
	"fmt"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"
)

const (
	descId               = "Terraform's internal resource ID. It is structured as \"`project_id`,`vpc_id`,`region`,`routing_table_id`,`route_id`\"."
	descProjectId        = "STACKIT Project ID to which the static route is associated."
	descVpcId            = "The VPC ID to which the static route is associated."
	descRoutingTableId   = "The routing table ID to which the static route is associated."
	descRouteId          = "The static route ID."
	descRegion           = "The region of the static route."
	descDestination      = "The destination of the static route."
	descDestinationValue = "CIDR value."
	descNexthop          = "The nexthop of the static route."
	descNexthopValue     = "Value of the nexthop"
	descLabels           = "Labels are key-value string pairs which can be attached to a resource container"
)

var (
	descDestinationType = fmt.Sprintf("CIDR type. %s Currently cidrv6 is unsupported.", utils.FormatPossibleValues("cidrv4", "cidrv6"))
	descNexthopType     = fmt.Sprintf("Type of the nexthop. %s Currently ipv6 is unsupported.", utils.FormatPossibleValues("blackhole", "internet", "ipv4", "ipv6"))
)
