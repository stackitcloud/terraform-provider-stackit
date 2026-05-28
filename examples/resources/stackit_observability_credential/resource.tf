resource "stackit_observability_credential" "example" {
  project_id  = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  instance_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  description = "Description of the credential."
}

resource "time_rotating" "rotate" {
  rotation_days = 80
}

resource "stackit_observability_credential" "example_rotate" {
  project_id  = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  instance_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  description = "Description of the credential."

  rotate_when_changed = {
    rotation = time_rotating.rotate.id
  }
}