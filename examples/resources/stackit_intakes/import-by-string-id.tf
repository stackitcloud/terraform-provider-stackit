# Only use the import statement, if you want to import an existing intake
import {
  to = stackit_intakes.import-example
  id = "${var.project_id},${var.region},${var.intake_id}"
}
