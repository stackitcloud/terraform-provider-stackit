variable "organization_id" {}
variable "name" {}
variable "ipv4_gateway" {}
variable "ipv4_nameserver_0" {}
variable "ipv4_nameserver_1" {}
variable "ipv4_prefix" {}
variable "ipv4_prefix_length" {}
variable "routed" {}
variable "label" {}
variable "service_account_mail" {}
variable "dhcp" {}

# no test candidate, just needed for the testing setup
resource "stackit_network_area" "network_area" {
  organization_id = var.organization_id
  name            = var.name
  labels = {
    "preview/routingtables" = "true"
  }
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

# no test candidate, just needed for the testing setup
resource "stackit_resourcemanager_project" "project" {
  parent_container_id = stackit_network_area.network_area.organization_id
  name                = var.name
  labels = {
    "networkArea" = stackit_network_area.network_area.network_area_id
  }
  owner_email = var.service_account_mail

  depends_on = [stackit_network_area_region.network_area_region]
}

resource "stackit_network" "network_prefix" {
  project_id = stackit_resourcemanager_project.project.project_id
  name       = var.name
  # ipv4_gateway       = var.ipv4_gateway != "" ? var.ipv4_gateway : null
  # no_ipv4_gateway    = var.ipv4_gateway != "" ? null : true
  ipv4_nameservers = [var.ipv4_nameserver_0, var.ipv4_nameserver_1]
  ipv4_prefix      = var.ipv4_prefix
  routed           = var.routed
  labels = {
    "acc-test" : var.label
  }
  dhcp = var.dhcp

  depends_on = [stackit_network_area_region.network_area_region]
}

resource "stackit_network" "network_prefix_length" {
  project_id = stackit_resourcemanager_project.project.project_id
  name       = var.name
  # no_ipv4_gateway    = true
  ipv4_nameservers   = [var.ipv4_nameserver_0, var.ipv4_nameserver_1]
  ipv4_prefix_length = var.ipv4_prefix_length
  routed             = var.routed
  labels = {
    "acc-test" : var.label
  }
  routing_table_id = stackit_routing_table.routing_table.routing_table_id

  depends_on = [stackit_network.network_prefix, stackit_network_area_region.network_area_region]
}

resource "stackit_routing_table" "routing_table" {
  organization_id = var.organization_id
  network_area_id = stackit_network_area.network_area.network_area_id
  name            = var.name

  depends_on = [stackit_network_area_region.network_area_region]
}
