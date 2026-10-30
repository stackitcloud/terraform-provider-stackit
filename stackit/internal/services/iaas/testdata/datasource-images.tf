variable "project_id" { type = string }

data "stackit_images" "all" {
  project_id = var.project_id
}
