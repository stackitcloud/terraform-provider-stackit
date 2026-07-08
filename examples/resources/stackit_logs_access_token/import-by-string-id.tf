# Only use the import statement, if you want to import an existing logs access token
# Note: The generated access token is only available upon creation.
# Since this attribute is not fetched from the API call, to prevent the conflicts, you need to add:
# lifecycle {
#   ignore_changes = [ lifetime ]
# }
import {
  to = stackit_logs_access_token.import-example
  id = "${var.project_id},${var.region},${var.logs_instance_id},${var.logs_access_token_id}"
}
