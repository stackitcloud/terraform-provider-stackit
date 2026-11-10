# Only use the import statement if you want to import an existing Container Registry artifactory
import {
  to = stackit_containerregistry_artifactory.this
  id = "${local.project_id},${local.region},ARTIFACTORY_ID"
}
