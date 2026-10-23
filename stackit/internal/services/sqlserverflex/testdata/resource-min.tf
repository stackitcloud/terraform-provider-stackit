variable "project_id" {}
variable "name" {}
variable "flavor_id" {}
variable "username" {}
variable "role" {}
variable "database_name" {}

resource "stackit_sqlserverflex_instance" "instance" {
  project_id = var.project_id
  name       = var.name
  flavor_id  = var.flavor_id
}

resource "stackit_sqlserverflex_user" "user" {
  project_id  = stackit_sqlserverflex_instance.instance.project_id
  instance_id = stackit_sqlserverflex_instance.instance.instance_id
  username    = var.username
  roles       = [var.role]
}

resource "stackit_sqlserverflex_database" "database" {
  project_id  = stackit_sqlserverflex_instance.instance.project_id
  instance_id = stackit_sqlserverflex_instance.instance.instance_id
  name        = var.database_name
  owner       = stackit_sqlserverflex_user.user.username
}

data "stackit_sqlserverflex_instance" "instance" {
  project_id  = var.project_id
  instance_id = stackit_sqlserverflex_instance.instance.instance_id
}

data "stackit_sqlserverflex_user" "user" {
  project_id  = var.project_id
  instance_id = stackit_sqlserverflex_instance.instance.instance_id
  user_id     = stackit_sqlserverflex_user.user.user_id
}

data "stackit_sqlserverflex_database" "database" {
  project_id  = stackit_sqlserverflex_instance.instance.project_id
  instance_id = stackit_sqlserverflex_instance.instance.instance_id
  name        = stackit_sqlserverflex_database.database.name
}
