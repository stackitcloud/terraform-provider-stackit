resource "stackit_git" "git" {
  project_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  name       = "git-example-instance"
}

resource "stackit_git" "git" {
  project_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  name       = "git-example-instance"
  acl = [
    "0.0.0.0/0"
  ]
  flavor = "git-100"
}

# Only use the import statement, if you want to import an existing git resource
import {
  to = stackit_git.import-example
  id = "${var.project_id},${var.git_instance_id}"
}