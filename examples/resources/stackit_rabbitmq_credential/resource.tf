resource "stackit_rabbitmq_credential" "example" {
  project_id  = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  instance_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
}

resource "time_rotating" "rotate" {
  rotation_days = 80
}

resource "stackit_rabbitmq_credential" "example" {
  project_id  = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  instance_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"

  rotate_when_changed = {
    rotation = time_rotating.rotate.id
  }
}

# Only use the import statement, if you want to import an existing rabbitmq credential
import {
  to = stackit_rabbitmq_credential.import-example
  id = "${var.project_id},${var.rabbitmq_instance_id},${var.rabbitmq_credential_id}"
}