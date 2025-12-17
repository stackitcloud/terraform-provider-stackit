variable "organization_id" {}

variable "name" {}

resource "stackit_network_area" "network_area" {
  organization_id = var.organization_id
  name            = var.name
}
