resource "stackit_redis_credential" "example" {
  project_id  = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  instance_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
}

# Only use the import statement, if you want to import an existing redis credential
import {
  to = stackit_redis_credential.import-example
  id = "${var.project_id},${var.redis_instance_id},${var.redis_credential_id}"
}