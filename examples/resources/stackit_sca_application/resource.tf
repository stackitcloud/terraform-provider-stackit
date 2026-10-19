resource "stackit_sca_application" "example" {
  project_id     = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  environment_id = "yyyyyyyy-yyyy-yyyy-yyyy-yyyyyyyyyyyy"
  display_name   = "test-application"

  containers = [
    {
      name       = "container-1"
      image      = "registry.example.com/test-application:v1.0.0"
      cpu_millis = 1000
      memory_mb  = 1024
      command    = ["/app/start.sh"]
      args       = ["--verbose", "--production"]

      env = {
        from_value = {
          "LOG_LEVEL" = "info"
          "NODE_ENV"  = "production"
        }
        from_secret_ref = {
          "DB_PASSWORD" = "prod-db-password"
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
    auto = {
      min_instances       = 2
      max_instances       = 10
      allow_scale_to_zero = true

      http_rule = {
        name        = "traffic-rule"
        concurrency = 10
      }
    }
  }
}