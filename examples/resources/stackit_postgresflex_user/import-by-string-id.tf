# Only use the import statement, if you want to import an existing postgresflex user
import {
  to = stackit_postgresflex_user.import-example
  id = "${var.project_id},${var.region},${var.postgres_instance_id},${var.user_id}"
}

# Only use the import statement, if you want to import an existing postgresflex user
# usually the imported user will not have a password after importing
# to be able to reference the imported users password, add reset as fifth parameter
# (this will reset the password of the imported user)
import {
  to = stackit_postgresflex_user.import-example
  id = "${var.project_id},${var.region},${var.postgres_instance_id},${var.user_id},reset"
}
