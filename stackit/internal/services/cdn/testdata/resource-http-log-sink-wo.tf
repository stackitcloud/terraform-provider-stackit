variable "project_id" {}
variable "regions" {}
variable "backend_http_type" {}
variable "backend_origin_url" {}
variable "logs_display_name" {}
variable "token_wo_version" {
  type    = number
  default = 1
}

resource "stackit_logs_instance" "logs" {
  project_id     = var.project_id
  region         = "eu01"
  display_name   = var.logs_display_name
  retention_days = 1
}

resource "stackit_logs_access_token" "logs_token" {
  project_id   = var.project_id
  instance_id  = stackit_logs_instance.logs.instance_id
  display_name = var.logs_display_name

  permissions = [
    "read", "write"
  ]
}

resource "stackit_cdn_distribution" "distribution" {
  project_id = var.project_id

  config = {
    regions = var.regions
    backend = {
      type       = var.backend_http_type
      origin_url = var.backend_origin_url
    }

    log_sink = {
      push_url = "https://${stackit_logs_instance.logs.ingest_otlp_url}"
      type     = "otlp"
      credentials = {
        type             = "bearer"
        token_wo         = stackit_logs_access_token.logs_token.access_token
        token_wo_version = var.token_wo_version
      }
    }
  }
}
