variable "organization_id" {}
variable "parent_container_id" {}
variable "owner_email" {}
variable "network_area_name" {}
variable "project_name" {}
variable "routing_table_name" {}
variable "region" {}
variable "display_name" {}
variable "plan_id" {}
variable "routing_type" {}
variable "az_tunnel1" {}
variable "az_tunnel2" {}
variable "local_asn" {}
variable "override_advertised_routes" {}
variable "label_key" {}
variable "label_value" {}
variable "network_config_prefix" {}

resource "stackit_network_area" "network_area" {
  organization_id = var.organization_id
  name            = var.network_area_name
  labels = {
    "preview/routingtables" = "true"
  }
}

resource "stackit_resourcemanager_project" "project" {
  parent_container_id = var.parent_container_id
  name                = var.project_name
  labels = {
    networkArea = stackit_network_area.network_area.network_area_id
  }
  owner_email = var.owner_email

  depends_on = [stackit_network_area_region.network_area_region]
}

resource "stackit_network_area_region" "network_area_region" {
  organization_id = var.organization_id
  network_area_id = stackit_network_area.network_area.network_area_id
  ipv4 = {
    network_ranges = [
      {
        prefix = "10.0.0.0/16"
      },
      {
        prefix = "10.2.2.0/24"
      }
    ]
    transfer_network = "10.1.2.0/24"
  }
}

resource "stackit_routing_table" "routing_table" {
  organization_id = stackit_network_area.network_area.organization_id
  network_area_id = stackit_network_area.network_area.network_area_id
  name            = var.routing_table_name
  depends_on      = [stackit_network_area_region.network_area_region]
}

resource "stackit_vpn_gateway" "gateway" {
  project_id   = stackit_resourcemanager_project.project.project_id
  region       = var.region
  display_name = var.display_name
  plan_id      = var.plan_id
  routing_type = var.routing_type

  availability_zones = {
    tunnel1 = var.az_tunnel1
    tunnel2 = var.az_tunnel2
  }

  network_config = {
    predefined_network_prefix = var.network_config_prefix
    routing_table_id          = stackit_routing_table.routing_table.routing_table_id
  }

  bgp = {
    local_asn                  = var.local_asn
    override_advertised_routes = var.override_advertised_routes
  }

  labels = var.label_key == "" ? {} : {
    (var.label_key) = var.label_value
  }
}
