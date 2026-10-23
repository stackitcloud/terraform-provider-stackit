variable "project_id" {}
variable "waf_configuration_name" {}

resource "stackit_alb_waf_configuration" "waf_instance" {
  project_id = var.project_id
  name       = var.waf_configuration_name
}

