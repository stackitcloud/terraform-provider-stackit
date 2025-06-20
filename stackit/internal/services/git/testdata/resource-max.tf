
variable "project_id" {}
variable "name" {}
variable "acl" {}
variable "flavor" {}

resource "stackit_git" "git" {
  project_id = var.project_id
  name = var.name
  acl = [
    var.acl
  ]
  flavor = var.flavor
}
