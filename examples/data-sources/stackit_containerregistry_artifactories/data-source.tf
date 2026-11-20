locals {
  project_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
}

data "stackit_containerregistry_artifactories" "this" {
  project_id = local.project_id
}
