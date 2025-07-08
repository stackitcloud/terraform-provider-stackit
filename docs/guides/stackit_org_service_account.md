---
page_title: "Creating projects in empty organization via Terraform"
---
# Creating projects in empty organization via Terraform

Consider the following situation: You're starting with an empty STACKIT organization and want to create projects 
in this organization using the `stackit_resourcemanager_project` resource. Unfortunately it's not possible to create
a service account on organization level which can be used for authentication in the STACKIT Terraform provider. 
The following steps will help you to get started:

1. Using the STACKIT portal, create a dummy project in your organization which will hold your service account, let's name it e.g. "dummy-service-account-project".
2. In this "dummy-service-account-project", create a service account. Create and save a service account key to use for authentication for the STACKIT Terraform provider later as described in the docs. Now copy the e-mail address of the service account you just created.
3. Here comes the important part: Navigate to your organization, open it and select "Access". Click on the "Grant access" button and paste the e-mail address of your service account. Be careful to grant the service account enough permissions to create projects in your organization, e.g. by assigning the "owner" role to it.

*This problem was brought up initially in [this](https://github.com/stackitcloud/terraform-provider-stackit/issues/855) issue on GitHub.*
