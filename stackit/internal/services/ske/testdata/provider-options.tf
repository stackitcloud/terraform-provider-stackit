variable "region" {}

data "stackit_ske_kubernetes_versions" "example" {
  region        = var.region
  version_state = "SUPPORTED"
}

data "stackit_ske_machine_image_versions" "example" {
  region        = var.region
  version_state = "SUPPORTED"
}
