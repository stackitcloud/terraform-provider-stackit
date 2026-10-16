package sca

const (
	descriptionResource = "SCA Application Resource."

	// Root fields
	descriptionApplicationId = "The application ID."
	descriptionContainers    = "List of containers to be run for every instance of the application."
	descriptionDisplayName   = "The application name."
	descriptionEnvironmentId = "Application environment ID."
	descriptionId            = "Terraform's internal resource ID. It is structured as \"`project_id`,`region`,`environment_id`,`application_id`\"."
	descriptionNetwork       = "Network configuration for an application. Defaults to a non-exposed application."
	descriptionProjectId     = "STACKIT project ID to which the application is associated."
	descriptionRegion        = "The STACKIT region where the application is deployed. If not defined, the provider region is used."
	descriptionStatus        = "Current status of the application."
	descriptionStopped       = "Indicates if the application resources are stopped. Defaults to `false`."
	descriptionUrls          = "Exposed URLs for the running application."

	// Container fields
	descriptionContainerName    = "The name of the container."
	descriptionContainerImage   = "The OCI-compliant container image reference, including the registry path and tag. Implicit public registry short names are supported. Note: The image must run in a non-privileged user namespace (non-root) to start successfully."
	descriptionContainerCpu     = "CPU limit for the container in millicores (e.g., 1000)."
	descriptionContainerMemory  = "Memory limit for the container in megabytes (e.g., 1024)."
	descriptionContainerCommand = "Overrides the container's default command."
	descriptionContainerArgs    = "Overrides the container's default arguments."
	descriptionContainerEnv     = "Environment variables to expose to the container."
	descriptionEnvFromValue     = "A key-value map of plaintext environment variables."
	descriptionEnvFromSecret    = "A key-value map of environment variables sourced from the SCA Environment, where the map value is the secret reference." //nolint:gosec // description for secret mapping

	// Network fields
	descriptionNetworkPublic        = "Public network configuration block for the application."
	descriptionNetworkPublicEnabled = "Whether the application should be accessible from the internet. Note: A default secure URL (HTTPS) will be generated."
	descriptionNetworkPublicPort    = "Port number where the application is listening in any container. The exposed URL traffic will be routed to this port unencrypted (HTTP). Note: In order to start properly, a non-privileged port number should be provided (e.g. 8080)."
	descriptionNetworkPublicAcl     = "The set of IPv4 address ranges, specified in CIDR notation, permitted for ingress traffic. Defaults to 0.0.0.0/0, allowing public internet access."

	// Scaling fields
	descriptionScaling             = "Horizontal scaling configuration for the application."
	descriptionScalingManual       = "Configuration for a static, fixed number of running instances. Note: Exactly one of `manual` or `auto` must be specified."
	descriptionScalingInstances    = "The exact number of application instances to run."
	descriptionScalingAuto         = "Configuration for dynamic autoscaling of the running instances. Note: Exactly one of `manual` or `auto` must be specified."
	descriptionScalingMinInstances = "The minimum number of application instances to run. Note: To allow the application to scale completely down to zero (0), use the `allow_scale_to_zero` flag."
	descriptionScalingMaxInstances = "The maximum number of application instances the autoscaler can provision."
	descriptionScalingScaleToZero  = "Enables scaling down to zero instances. When enabled and the defined rules evaluate to 0, the application enters an idle state. Defaults to `false`. Note: Applications scaled to zero require additional spin-up time (a cold start) for the first rule that evaluates positively."
	descriptionScalingHttpRule     = "An HTTP-based scaling rule triggered by incoming traffic to the exposed URLs. Note: At least one of `http_rule` or `native_rules` must be specified when configuring auto-scaling. Also, when the application is scaled to zero (see: `allow_scale_to_zero`), all incoming requests will be held until the application starts listening to the port. Therefore, when no traffic is received in a period of time, the application will scale down to zero."
	descriptionRuleName            = "The name of the scaling rule."
	descriptionRuleConcurrency     = "The target number of concurrent HTTP requests per instance. Note: At least one of `concurrency` or `rps` must be specified."
	descriptionRuleRps             = "The target number of requests per second (RPS) per instance. Note: At least one of `concurrency` or `rps` must be specified."
	descriptionScalingNativeRules  = "Custom scaling rules based on native KEDA scalers. Refer to the official KEDA documentation for specific trigger configuration details. Note: At least one of `http_rule` or `native_rules` must be specified when configuring auto-scaling."
	descriptionRuleTrigger         = "The KEDA scaler trigger type (e.g., `rabbitmq`)."
	descriptionRuleParameters      = "A key-value map of scaler-specific parameters, as defined in the KEDA Trigger Specification section."
	descriptionRuleSecretsMapping  = "Secret mappings for the TriggerAuthentication resource, key must be listed in the Authentication Parameters scaler documentation. Its value will refer to the secret reference to the SCA Environment secrets."
)
