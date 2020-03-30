---
name: Common Terraform Setup
menu: Commands
route: /commands/common-terraform-setup
---

# Common Terraform Setup

With commands other than [setup](setup) and [release](release), the release is downloaded and unpacked
(including the exact Terraform image, terraform providers and terraform modules). The config contianer
also returns the [backend config](https://www.terraform.io/docs/backends/index.html) and the
[terraform workspace](https://www.terraform.io/docs/state/workspaces.html) is configured.

If it doesn't already exist then an empty backend [partial configuration](https://www.terraform.io/docs/backends/config.html#partial-configuration) is written to `infra/terraform.tf` with the backend type returned from the config container:

```terraform
terraform {
  backend "RETURNED_BACKEND_TYPE" {}
}
```

Terraform is then run to complete backend configuration based on the config keys and values returned from the config container - similar to the following:

```shell
terraform init \
    -get=false \
    -get-plugins=false \
    -backend-config="key1=value1" \
    -backend-config="key2=value2"
```

Following this the workspace is initialised (if neccessary) and selected, similar to:

```shell
if terraform workspace list | grep -q '\bENV\b'; then
    terraform workspace select ENV
else
    terraform workspace new ENV
fi
```

