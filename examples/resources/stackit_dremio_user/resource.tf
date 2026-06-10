resource "stackit_dremio_user" "example" {
  project_id    = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  region        = "eu01"
  instance_id   = "example-instance-id"

  description = "STACKIT Terraform example"
  email = "example@example.com"
  first_name = "Test"
  last_name = "User"
  name = "testUser"
}

import {
  to = stackit_dremio_user.import_example
  id = "${var.project_id},${var.region},${var.instance_id},${var.user_id}"
}