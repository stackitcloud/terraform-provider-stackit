resource "stackit_logme_instance" "example" {
  project_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  name       = "example-instance"
  version    = "2"
  plan_name  = "stackit-logme2-1.2.50-replica"
  parameters = {
    sgw_acl = "193.148.160.0/19,45.129.40.0/21,45.135.244.0/22"
  }
}

# Only use the import statement, if you want to import an existing logme instance
import {
  to = stackit_logme_instance.import-example
  id = "${var.project_id},${var.logme_instance_id}"
}