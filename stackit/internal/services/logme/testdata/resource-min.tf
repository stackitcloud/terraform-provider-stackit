variable "project_id" {}
variable "name" {}
variable "plan_name" {}
variable "logme_version" {}

resource "stackit_logme_instance" "instance" {
  project_id = var.project_id
  name       = var.name
  plan_name  = var.plan_name
  version    = var.logme_version
}

resource "stackit_logme_credential" "credential" {
  project_id  = stackit_logme_instance.instance.project_id
  instance_id = stackit_logme_instance.instance.instance_id
}


data "stackit_logme_instance" "instance" {
  project_id  = stackit_logme_instance.instance.project_id
  instance_id = stackit_logme_instance.instance.instance_id
}

data "stackit_logme_credential" "credential" {
  project_id    = stackit_logme_credential.credential.project_id
  instance_id   = stackit_logme_credential.credential.instance_id
  credential_id = stackit_logme_credential.credential.credential_id
}
