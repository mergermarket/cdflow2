---
name: Deploy
menu: Commands
route: /commands/deploy
---

# Deploy

## Usage

`cdflow2 [ GLOBALOPTS ] deploy [ OPTS ] ENV VERSION`

See [usage](./usage) for global options.

### Arguments:

`ENV`
: The environment being deployed to.

`VERSION`
: The version being deployed (must match what was released).

### Options:

`--plan-only` | `-p`
: Create the terraform plan only, don't apply.

`--new-state` | `-n`
: Allow run without a pre-existing tfstate file.

## Description

Terraform is configured as described in [common terraform setup](common-terraform-setup.md), followed by commands
equivalent to:

```shell-session
$ cd infra
$ terraform plan \
    -input=false \
    -var-file release-metadata-VERSION.json \
    -var-file config/ENV.json \
    -out=plan-TIMESTAMP
$ 
$ terraform apply \
    -input=false \
    plan-TIMESTAMP
```

## First Deployment to an Environment

The [Terraform State](https://www.terraform.io/docs/language/state/index.html) is used to track
resources managed by Terraform. It ensures that Terraform is operating on the same resources between
runs in a given environment. It is therefore important that the state file exists to avoid losing
track of resources. Unfortuntely if you change something that is part of the location of the state
file (e.g. the component name, derived by default from the repo name) then it isn't possible to tell
this is the case rather than the reason for no state file 

When running `terraform apply` for the first time in an environment
 the statefile should not exist. On
subsequent runs the statefile should

The `--new-state` or `-n` flag is required for the first deployment into a particular
environment, but then must be removed for subsequent deployments. This is a safety
feature that ensures you do not lose track of your tfstate (e.g. if the component name
changes but you haven't moved the statefile accordingly).
