data "stackit_service_accounts" "all_sas" {
  project_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
}

data "stackit_service_accounts" "sas_default_suffix" {
  project_id   = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  email_suffix = "@sa.stackit.cloud"
}

data "stackit_service_accounts" "sas_default_suffix_sort_asc" {
  project_id     = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  email_suffix   = "@sa.stackit.cloud"
  sort_ascending = true
}

data "stackit_service_accounts" "sas_ske_regex" {
  project_id  = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  email_regex = ".*@ske\\.sa\\.stackit\\.cloud$"
}

data "stackit_service_accounts" "sas_ske_suffix" {
  project_id   = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  email_suffix = "@ske.sa.stackit.cloud"
}