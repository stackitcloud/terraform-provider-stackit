resource "stackit_sqlserverflex_user" "example" {
  project_id  = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  instance_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  username    = "username"
  roles       = ["role"]
}

resource "time_rotating" "rotate" {
  rotation_days = 80
}

resource "stackit_sqlserverflex_user" "example_rotate" {
  project_id  = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  instance_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  username    = "username"
  roles       = ["role"]

  rotate_when_changed = {
    rotation = time_rotating.rotate.id
  }
}