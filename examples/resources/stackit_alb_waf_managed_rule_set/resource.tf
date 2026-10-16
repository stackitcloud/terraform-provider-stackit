resource "stackit_alb_waf_managed_rule_set" "example" {
  project_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  name       = "example-managed-rule-set"
  type       = "TYPE_OWASP_CRS"
}

# Set individual rules to a different mode, keyed by rule ID. Only the
# listed rules are managed; a rule removed from the map goes back to
# MODE_ENABLED.
resource "stackit_alb_waf_managed_rule_set" "with_rule_modes" {
  project_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  name       = "example-managed-rule-set-modes"
  type       = "TYPE_OWASP_CRS"
  rule_modes = {
    "911100" = "MODE_LOG_ONLY" # allow PUT and DELETE through method enforcement
    "920450" = "MODE_DISABLED"
  }
}
