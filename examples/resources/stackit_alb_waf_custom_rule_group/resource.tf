resource "stackit_alb_waf_custom_rule_group" "example" {
  project_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  name       = "example-custom-rule-group"
  rules = [
    {
      description = "My custom rule group"
      behavior = {
        action = "ACTION_DENY"
        log    = true
        log_msg = "Some custom notification message string"
      }
      conditions = [
        {
          operator = {
            type  = "OPERATOR_BEGINS_WITH"
            value = "allowed objects"
          }
          transformations = [
            "TRANSFORMATION_LOWERCASE"
          ]
          variable = {
            type  = "VARIABLE_REQUEST_HEADERS"
            value = "Host"
          }
        }
      ]
    }
  ]
}
