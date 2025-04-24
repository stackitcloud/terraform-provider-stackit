
variable "project_id" {}
variable "name" {}

resource "stackit_git" "git" {
  project_id = var.project_id
  name = var.name
}
