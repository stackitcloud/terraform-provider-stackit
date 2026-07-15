resource "stackit_modelexperiments_instance" "example" {
  project_id                   = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  region                       = "eu01"
  name                         = "Example name"
  description                  = "Example description"
  deleted_experiment_retention = "30d"
  labels = {
    label = "Example label"
  }
}

resource "time_rotating" "rotate" {
  rotation_days = 80
}

resource "stackit_modelexperiments_token" "token" {
  project_id   = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  name         = "Example token nane"
  region       = "eu01"
  instance_id  = stackit_modelexperiments_instance.example.instance_id
  description  = "Example token description"
  ttl_duration = "1h"
  labels = {
    label = "Example label"
  }
  rotate_when_changed = {
    rotation = time_rotating.rotate.id
  }
}