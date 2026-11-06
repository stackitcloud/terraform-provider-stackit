locals {
  project_id     = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  artifactory_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
}

data "stackit_containerregistry_artifactory" "this" {
  project_id     = local.project_id
  artifactory_id = local.artifactory_id
}
