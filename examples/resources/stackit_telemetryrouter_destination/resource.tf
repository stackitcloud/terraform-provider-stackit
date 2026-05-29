resource "stackit_telemetryrouter_destination" "s3" {
  project_id   = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  instance_id  = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  display_name = "s3-destination"
  description  = "S3 destination description"
  config = {
    config_type = "S3"
    filter = {
      attributes = [
        {
          key     = "key"
          level   = "logRecord"
          matcher = "!="
          values  = ["test1", "test2"]
        },
        {
          key     = "key2"
          level   = "resource"
          matcher = "="
          values  = ["test3"]
        }
      ]
    }
    s3 = {
      bucket = "test"
      access_key = {
        id     = "id"
        secret = "secret"
      }
      endpoint = "http://localhost:8160"
    }
  }
}

resource "stackit_telemetryrouter_destination" "otlp" {
  project_id   = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  instance_id  = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  display_name = "s3-destination"
  description  = "S3 destination description"
  config = {
    config_type = "OpenTelemetry"
    filter = {
      attributes = [
        {
          key     = "key"
          level   = "logRecord"
          matcher = "!="
          values  = ["test1", "test2"]
        },
        {
          key     = "key2"
          level   = "resource"
          matcher = "="
          values  = ["test3"]
        }
      ]
    }
    opentelemetry = {
      basic_auth = {
        username = "user2"
        password = "pass2"
      }
      # bearer_token = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiYWRtaW4iOnRydWUsImlhdCI6MTUxNjIzOTAyMn0.KMUFsIDTnFmyG3nMiGM6H9FNFUROf3wh7SmqJp-QV30"
      uri = "http://localhost:8116"
    }
  }
}

# Only use the import statement, if you want to import an existing TelemetryRouter destination
import {
  to = stackit_telemetryrouter_destination.import-example
  id = "${var.project_id},${var.region},${var.telemetryrouter_instance_id},${var.destination_id}"
}
