resource "stackit_mariadb_credential" "example" {
  project_id  = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  instance_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
}

resource "time_rotating" "rotate" {
  rotation_days = 80
}

resource "stackit_mariadb_credential" "example_rotate" {
  project_id  = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  instance_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"

  rotate_when_changed = {
    rotation = time_rotating.rotate.id
  }
}

# Only use the import statement, if you want to import an existing mariadb credential
import {
  to = stackit_mariadb_credential.import-example
  id = "${var.project_id},${var.mariadb_instance_id},${var.mariadb_credential_id}"
}
