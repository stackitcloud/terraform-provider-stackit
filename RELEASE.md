# Release

## Release cycle

A release should be created at least every 2 weeks. 

## Release creation

> [!IMPORTANT]
> Consider informing / syncing with the team before creating a new release.

1. Check out latest main branch on your machine
2. Create git tag: `git tag vX.X.X`
3. Push the git tag: `git push origin --tags`
4. The [release pipeline](https://github.com/stackitcloud/terraform-provider-stackit/actions/workflows/release.yaml) will build the release and publish it on GitHub
5. Ensure the release was created properly using the 
    - [GitHub releases page](https://github.com/stackitcloud/terraform-provider-stackit/releases)
    - [Terraform registry](https://registry.terraform.io/providers/stackitcloud/stackit/latest)

## Troubleshooting

In case the release only shows up as a draft release in the Terraform registry, try to temporariliy declare the release as a pre-release in GitHub and then revert it it immediately.

