resource "stackit_affinity_group" "example" {
  project_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  name       = "example-affinity-group-name"
  policy     = "hard-anti-affinity"
}

# Only use the import statement, if you want to import an existing affinity group
import {
  to = stackit_affinity_group.import-example
  id = "${var.project_id},${var.region},${var.affinity_group_id}"
}