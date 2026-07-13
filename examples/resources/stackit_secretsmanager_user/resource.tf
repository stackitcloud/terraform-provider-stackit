resource "stackit_secretsmanager_user" "example" {
  project_id    = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  instance_id   = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  description   = "Example user"
  write_enabled = false
}

resource "time_rotating" "rotate" {
  rotation_days = 80
}

resource "stackit_secretsmanager_user" "example_rotate" {
  project_id    = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  instance_id   = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  description   = "Example user"
  write_enabled = false

  rotate_when_changed = {
    rotation = time_rotating.rotate.id
  }
}
