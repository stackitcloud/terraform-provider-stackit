resource "stackit_workflows_dag_bundle" "git" {
  project_id  = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  region      = "eu01"
  instance_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"

  name   = "main-dags"
  type   = "git"
  url    = "https://git.example.com/my-org/my-dags.git"
  branch = "main"
  subdir = "dags/"
  auth = {
    type     = "basic"
    username = "git-user"
    password = "personal-access-token"
  }
}

resource "stackit_workflows_dag_bundle" "git_public" {
  project_id  = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  region      = "eu01"
  instance_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"

  name   = "public-dags"
  type   = "git"
  url    = "https://github.com/example/public-airflow-dags.git"
  branch = "main"
  auth = {
    type = "none"
  }
}

resource "stackit_workflows_dag_bundle" "s3" {
  project_id  = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  region      = "eu01"
  instance_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"

  name             = "backup-dags"
  type             = "s3"
  bucket_name      = "my-airflow-dags"
  prefix           = "dags/"
  refresh_interval = 60
  s3_auth = {
    type              = "access_key"
    access_key_id     = "AKIA..."
    secret_access_key = "shhh"
  }
}

# Only use the import statement, if you want to import an existing Workflows DAG bundle.
# Sensitive fields (password, secret_access_key) cannot be imported — they are never returned by the API.
import {
  to = stackit_workflows_dag_bundle.import-example
  id = "${var.project_id},${var.region},${var.workflows_instance_id},${var.bundle_name}"
}
