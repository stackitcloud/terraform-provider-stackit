# Only use the import statement, if you want to import an existing git resource
import {
  to = stackit_git.import-example
  id = "${var.project_id},${var.git_instance_id}"
}
