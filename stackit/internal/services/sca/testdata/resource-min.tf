variable "project_id" {
  type = string
}

variable "region" {
  type = string
}

variable "app_name" {
  type = string
}

variable "stopped" {
  type = bool
}

variable "instances" {
  type = number
}

resource "stackit_sca_application" "app" {
  project_id     = var.project_id
  region         = var.region
  environment_id = var.project_id # As in alpha, environments are WIP, so equaling to project_id uses default environment
  display_name   = var.app_name
  stopped        = var.stopped

  containers = [
    {
      name       = "nginx"
      image      = "nginxinc/nginx-unprivileged:latest" # SCA Runs on a non-privileged environment
      cpu_millis = 1000
      memory_mb  = 512
      env = {
        from_value = {
          "TEST_KEY" = "test_value"
        }
      }
    }
  ]

  network = {
    public = {
      enabled = true
      port    = 8080
    }
  }

  scaling = {
    manual = {
      instances = var.instances
    }
  }
}