---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "stackit_argus_scrapeconfig Data Source - stackit"
subcategory: ""
description: |-
  Argus scrape config data source schema. Must have a region specified in the provider configuration.
  !> The stackit_argus_scrapeconfig data source has been deprecated and will be removed after February 26th 2025. Please use stackit_observability_scrapeconfig instead, which offers the exact same functionality.
---

# stackit_argus_scrapeconfig (Data Source)

Argus scrape config data source schema. Must have a `region` specified in the provider configuration.

!> The `stackit_argus_scrapeconfig` data source has been deprecated and will be removed after February 26th 2025. Please use `stackit_observability_scrapeconfig` instead, which offers the exact same functionality.

## Example Usage

```terraform
data "stackit_argus_scrapeconfig" "example" {
  project_id  = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  instance_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  job_id      = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
}
```

<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `instance_id` (String) Argus instance ID to which the scraping job is associated.
- `name` (String) Specifies the name of the scraping job
- `project_id` (String) STACKIT project ID to which the scraping job is associated.

### Read-Only

- `basic_auth` (Attributes) A basic authentication block. (see [below for nested schema](#nestedatt--basic_auth))
- `id` (String) Terraform's internal data source. ID. It is structured as "`project_id`,`instance_id`,`name`".
- `metrics_path` (String) Specifies the job scraping url path.
- `saml2` (Attributes) A SAML2 configuration block. (see [below for nested schema](#nestedatt--saml2))
- `sample_limit` (Number) Specifies the scrape sample limit.
- `scheme` (String) Specifies the http scheme.
- `scrape_interval` (String) Specifies the scrape interval as duration string.
- `scrape_timeout` (String) Specifies the scrape timeout as duration string.
- `targets` (Attributes List) The targets list (specified by the static config). (see [below for nested schema](#nestedatt--targets))

<a id="nestedatt--basic_auth"></a>
### Nested Schema for `basic_auth`

Read-Only:

- `password` (String, Sensitive) Specifies basic auth password.
- `username` (String) Specifies basic auth username.


<a id="nestedatt--saml2"></a>
### Nested Schema for `saml2`

Read-Only:

- `enable_url_parameters` (Boolean) Specifies if URL parameters are enabled


<a id="nestedatt--targets"></a>
### Nested Schema for `targets`

Read-Only:

- `labels` (Map of String) Specifies labels.
- `urls` (List of String) Specifies target URLs.
