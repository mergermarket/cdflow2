---
name: Common Terraform Setup
menu: Commands
route: /commands/common-terraform-setup
---

# Common Terraform Setup

For commands that run terraform against an environment like [deploy](deploy) there is a common
process for setting up terraform that is performed:

* The release is downloaded and unpacked, including the release data and the exact Terraform [image](https://registry.hub.docker.com/r/hashicorp/terraform), [.terraform.lock.hcl](https://www.terraform.io/docs/language/dependency-lock.html) (see below) & [modules](https://www.terraform.io/docs/modules/index.html) in order that nothing changes as your release is promoted through the pipeline.
* The [terraform backend](https://www.terraform.io/docs/backends/index.html) is configured using config returned from the config container, in order to load and save your Terraform [state](https://www.terraform.io/docs/state/index.html).
* The [terraform workspace](https://www.terraform.io/docs/state/workspaces.html) is initialised and selected for the environment you are using.

## Backend

If it doesn't already exist then an empty backend
[partial configuration](https://www.terraform.io/docs/backends/config.html#partial-configuration)
is written to `infra/terraform.tf` with the backend type returned from the config container:

```hcl
terraform {
  backend "RETURNED_BACKEND_TYPE" {}
}
```

Terraform is then run to complete backend configuration based on the config keys and values returned from the config container - similar to the following:

```shell-session
$ terraform init \
    -get=false \
    -backend-config="key1=value1" \
    -backend-config="key2=value2"
```

## Workspace

The [terraform workspace](https://www.terraform.io/docs/state/workspaces.html) is initialised (if
neccessary) and selected, similar to:

```shell
$ if terraform workspace list | grep -q '\bENV\b'; then
    terraform workspace select ENV
else
    terraform workspace new ENV
fi
```

## Dependency Lock File

`cdflow2` uses the [`.terraform.lock.hcl` file](https://www.terraform.io/docs/language/dependency-lock.html) to ensure that provider versions do not change as a release is promoted through your pipeline. If there is aready an `infra/.terraform.lock.hcl` file committed then that will be used. Otherwise `cdflow2` will save the one created during the release step in the release archive and ensure it is in place for each Terraform operation (e.g. during deploy).
