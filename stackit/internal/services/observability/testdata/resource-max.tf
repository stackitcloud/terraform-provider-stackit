
variable "project_id" {}

variable "alertgroup_name" {}
variable "alert_rule_name" {}
variable "alert_rule_expression" {}
variable "alert_for_time" {}
variable "alert_label" {}
variable "alert_annotation" {}
variable "alert_interval" {}

variable "instance_name" {}
variable "plan_name" {}
variable "logs_retention_days" {}
variable "traces_retention_days" {}
variable "metrics_retention_days" {}
variable "metrics_retention_days_5m_downsampling" {}
variable "metrics_retention_days_1h_downsampling" {}
variable "instance_acl_1" {}
variable "instance_acl_2" {}
variable "receiver_name" {}
variable "auth_identity" {}
variable "auth_password" {}
variable "auth_username" {}
variable "email_from" {}
variable "email_send_resolved" {}
variable "smart_host" {}
variable "email_to" {}
variable "opsgenie_api_key" {}
variable "opsgenie_api_tags" {}
variable "opsgenie_api_url" {}
variable "opsgenie_priority" {}
variable "opsgenie_send_resolved" {}
variable "webhook_configs_url" {}
variable "ms_teams" {}
variable "google_chat" {}
variable "webhook_configs_send_resolved" {}
variable "group_by" {}
variable "group_interval" {}
variable "group_wait" {}
variable "repeat_interval" {}
variable "resolve_timeout" {}
variable "smtp_auth_identity" {}
variable "smtp_auth_password" {}
variable "smtp_auth_username" {}
variable "smtp_from" {}
variable "smtp_smart_host" {}
variable "match" {}
variable "match_regex" {}
variable "matchers" {}
variable "continue" {}

variable "credential_description" {}

variable "logalertgroup_name" {}
variable "logalertgroup_alert" {}
variable "logalertgroup_expression" {}
variable "logalertgroup_for_time" {}
variable "logalertgroup_label" {}
variable "logalertgroup_annotation" {}
variable "logalertgroup_interval" {}

variable "scrapeconfig_name" {}
variable "scrapeconfig_metrics_path" {}
variable "scrapeconfig_targets_url_1" {}
variable "scrapeconfig_targets_url_2" {}
variable "scrapeconfig_label" {}
variable "scrapeconfig_interval" {}
variable "scrapeconfig_limit" {}
variable "scrapeconfig_enable_url_params" {}
variable "scrapeconfig_scheme" {}
variable "scrapeconfig_timeout" {}
variable "scrapeconfig_auth_username" {}
variable "scrapeconfig_auth_password" {}

resource "stackit_observability_alertgroup" "alertgroup" {
  project_id  = var.project_id
  instance_id = stackit_observability_instance.instance.instance_id
  name        = var.alertgroup_name
  rules = [
    {
      alert      = var.alert_rule_name
      expression = var.alert_rule_expression
      for        = var.alert_for_time
      labels = {
        label1 = var.alert_label
      },
      annotations = {
        annotation1 = var.alert_annotation
      }
    }
  ]
  interval = var.alert_interval
}

resource "stackit_observability_credential" "credential" {
  project_id  = var.project_id
  instance_id = stackit_observability_instance.instance.instance_id
  description = var.credential_description
}

resource "stackit_observability_instance" "instance" {
  project_id = var.project_id
  name       = var.instance_name
  plan_name  = var.plan_name

  logs_retention_days                    = var.logs_retention_days
  traces_retention_days                  = var.traces_retention_days
  metrics_retention_days                 = var.metrics_retention_days
  metrics_retention_days_5m_downsampling = var.metrics_retention_days_5m_downsampling
  metrics_retention_days_1h_downsampling = var.metrics_retention_days_1h_downsampling
  acl                                    = [var.instance_acl_1, var.instance_acl_2]

  // alert config
  alert_config = {
    receivers = [
      {
        name = var.receiver_name
        email_configs = [
          {
            auth_identity = var.auth_identity
            auth_password = var.auth_password
            auth_username = var.auth_username
            from          = var.email_from
            smart_host    = var.smart_host
            to            = var.email_to
            send_resolved = var.email_send_resolved
          }
        ]
        opsgenie_configs = [
          {
            api_key       = var.opsgenie_api_key
            tags          = var.opsgenie_api_tags
            api_url       = var.opsgenie_api_url
            priority      = var.opsgenie_priority
            send_resolved = var.opsgenie_send_resolved
          }
        ]
        webhooks_configs = [
          {
            url           = var.webhook_configs_url
            ms_teams      = var.ms_teams
            google_chat   = var.google_chat
            send_resolved = var.webhook_configs_send_resolved
          }
        ]
      },
    ],

    route = {
      group_by        = [var.group_by]
      group_interval  = var.group_interval
      group_wait      = var.group_wait
      receiver        = var.receiver_name
      repeat_interval = var.repeat_interval
      routes = [
        {
          group_by        = [var.group_by]
          group_interval  = var.group_interval
          group_wait      = var.group_wait
          receiver        = var.receiver_name
          repeat_interval = var.repeat_interval
          continue        = var.continue
          match = {
            match1 = var.match
          }
          match_regex = {
            match_regex1 = var.match_regex
          }
          matchers = [
            var.matchers
          ]
        }
      ]
    },

    global = {
      opsgenie_api_key   = var.opsgenie_api_key
      opsgenie_api_url   = var.opsgenie_api_url
      resolve_timeout    = var.resolve_timeout
      smtp_auth_identity = var.smtp_auth_identity
      smtp_auth_password = var.smtp_auth_password
      smtp_auth_username = var.smtp_auth_username
      smtp_from          = var.smtp_from
      smtp_smart_host    = var.smtp_smart_host
    }
  }
}

resource "stackit_observability_logalertgroup" "logalertgroup" {
  project_id  = var.project_id
  instance_id = stackit_observability_instance.instance.instance_id
  name        = var.logalertgroup_name
  rules = [
    {
      alert      = var.logalertgroup_alert
      expression = var.logalertgroup_expression
      for        = var.logalertgroup_for_time
      labels = {
        label1 = var.logalertgroup_label
      },
      annotations = {
        annotation1 = var.logalertgroup_annotation
      }
    }
  ]
  interval = var.logalertgroup_interval
}

resource "stackit_observability_scrapeconfig" "scrapeconfig" {
  project_id   = var.project_id
  instance_id  = stackit_observability_instance.instance.instance_id
  name         = var.scrapeconfig_name
  metrics_path = var.scrapeconfig_metrics_path

  targets = [{
    urls = [var.scrapeconfig_targets_url_1, var.scrapeconfig_targets_url_2]
    labels = {
      label1 = var.scrapeconfig_label
    }
  }]
  scheme         = var.scrapeconfig_scheme
  scrape_timeout = var.scrapeconfig_timeout
  basic_auth = {
    username = var.scrapeconfig_auth_username
    password = var.scrapeconfig_auth_password
  }
  scrape_interval = var.scrapeconfig_interval
  sample_limit    = var.scrapeconfig_limit
  saml2 = {
    enable_url_parameters = var.scrapeconfig_enable_url_params
  }

}
