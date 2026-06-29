resource "stackit_modelexperiments_instance" "example" {
  project_id                   = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  region                       = "eu01"
  display_name                 = "example name"
  description                  = "Example description"
  deleted_experiment_retention = "30d"
  labels = {
    label = "example label"
  }
}