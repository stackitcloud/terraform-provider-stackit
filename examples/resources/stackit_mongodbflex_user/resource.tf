resource "stackit_mongodbflex_user" "example" {
  project_id  = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  instance_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  username    = "username"
  roles       = ["role"]
  database    = "database"
}

resource "time_rotating" "rotate" {
  rotation_days = 80
}

resource "stackit_mongodbflex_user" "example_rotate" {
  project_id  = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  instance_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  username    = "username"
  roles       = ["role"]
  database    = "database"

  rotate_when_changed = {
    rotation = time_rotating.rotate.id
  }
}

# Only use the import statement, if you want to import an existing mongodbflex user
import {
  to = stackit_mongodbflex_user.import-example
  id = "${var.project_id},${var.region},${var.instance_id},${user_id}"
}
