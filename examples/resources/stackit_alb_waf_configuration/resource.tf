resource "stackit_alb_waf_managed_rule_set" "example" {
  project_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  name       = "example-managed-rule-set"
  type       = "TYPE_OWASP_CRS"
}

resource "stackit_alb_waf_custom_rule_group" "example" {
  project_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  name       = "example-custom-rule-group"
  rules = [
    {
      behavior = {
        action = "ACTION_DENY"
      }
      conditions = [
        {
          operator = {
            type = "OPERATOR_VALIDATE_UTF8_ENCODING"
          }
          variable = {
            type = "VARIABLE_REQUEST_HEADERS"
          }
        }
      ]
    }
  ]
}

resource "stackit_alb_waf_configuration" "example" {
  project_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  name       = "example-waf-configuration"

  managed_rule_set_name  = stackit_alb_waf_managed_rule_set.example.name
  custom_rule_group_name = stackit_alb_waf_custom_rule_group.example.name

  labels = {
    "key" = "value"
  }
}
