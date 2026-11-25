# Only use the import statement, if you want to import an existing service enablement
# Make sure the service ID is part of the possible values for the service id field.
# Else importing the service enablement into Terraform might cause problems.
import {
  to = stackit_service_enablement.import-example
  id = "${var.project_id},${var.region},${var.service_id}"
}
