resource "stackit_alb_waf_managed_rule_set" "example" {
  project_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  name       = "example-managed-rule-set"
  type       = "TYPE_OWASP_CRS"
}

# Set individual rules to a different mode, keyed by group ID and rule ID.
# Only the listed rules are managed; a rule removed from configured_groups goes back to
# MODE_ENABLED. The state of all rules is available in groups.
resource "stackit_alb_waf_managed_rule_set" "with_rule_modes" {
  project_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  name       = "example-managed-rule-set-modes"
  type       = "TYPE_OWASP_CRS"
  configured_groups = {
    "911" = {
      rules = {
        "911100" = { mode = "MODE_LOG_ONLY" } # allow PUT and DELETE through method enforcement
      }
    }
    "920" = {
      rules = {
        "920450" = { mode = "MODE_DISABLED" }
      }
    }
  }
}
