resource "stackit_cdn_distribution" "example_distribution" {
  project_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  config = {
    backend = {
      type       = "http"
      origin_url = "https://mybackend.onstackit.cloud"
      geofencing = {
        "https://mybackend.onstackit.cloud" = ["DE"]
      }
    }
    regions           = ["EU", "US", "ASIA", "AF", "SA"]
    blocked_countries = ["DE", "AT", "CH"]

    optimizer = {
      enabled = true
    }
  }
}

resource "stackit_cdn_distribution" "example_bucket_distribution" {
  project_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  config = {
    backend = {
      type       = "bucket"
      bucket_url = "https://my-test.object.storage.eu01.onstackit.cloud"
      region     = "eu01"

      # Credentials are required for bucket backends
      # It is strongly recommended to use variables for secrets
      credentials = {
        access_key_id     = var.bucket_access_key
        secret_access_key = var.bucket_secret_key
      }
    }
    regions           = ["EU", "US"]
    blocked_countries = ["CN", "RU"]

    optimizer = {
      enabled = false
    }

    redirects = {
      rules = [
        {
          description          = "test redirect"
          enabled              = true
          rule_match_condition = "ANY"
          status_code          = 302
          target_url           = "https://stackit.de/"
          matchers = [
            {
              values                = ["*/otherPath/"]
              value_match_condition = "ANY"
            }
          ]
        }
      ]
    }

    # WAF Configuration
    # 
    # Precedence Hierarchy: Specific Rules > Groups > Collections
    # In this example, the entire "@builtin/crs/request" collection is ENABLED.
    # However, because specific Rule IDs have a higher precedence, the rule 
    # "@builtin/crs/request/942151" is explicitly DISABLED, overriding the collection setting.
    # 
    # To view all available collections, groups, and rules, consult the API documentation:
    # https://internal-docs.api.eu01.stackit.cloud/documentation/cdn/version/v1#tag/WAF/operation/ListWafCollections
    waf = {
      mode                          = "ENABLED"
      type                          = "PREMIUM"
      paranoia_level                = "L1"
      allowed_http_versions         = ["HTTP/1.0", "HTTP/1.1"]
      allowed_http_methods          = ["GET"]
      allowed_request_content_types = ["text/plain"]

      # Collections
      enabled_rule_collection_ids  = ["@builtin/crs/request"]
      disabled_rule_collection_ids = []
      log_only_rule_collection_ids = ["@builtin/crs/response"]

      # Groups
      enabled_rule_group_ids  = []
      disabled_rule_group_ids = []
      log_only_rule_group_ids = []

      # Specific Rules (Highest Precedence)
      enabled_rule_ids  = ["@builtin/crs/request/913100"]
      disabled_rule_ids = ["@builtin/crs/request/942151"]
      log_only_rule_ids = ["@builtin/crs/response/954120"]
    }
  }
}

# Only use the import statement, if you want to import an existing cdn distribution
import {
  to = stackit_cdn_distribution.import-example
  id = "${var.project_id},${var.distribution_id}"
}