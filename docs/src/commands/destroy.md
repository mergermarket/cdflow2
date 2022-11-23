---
name: Destroy
menu: Commands
route: /commands/destroy
---

# Destroy

## Usage

`cdflow2 [ GLOBALOPTS ] destroy [ OPTS ] ENV VERSION`

See [usage](./usage) for global options.

### Arguments:

`ENV`
: The environment containing the infrastructure being destroyed.

`VERSION`
: The version to destroy (must match a pre-existing release).

### Options:

`--plan-only` | `-p`
: Generate an execution plan only, don't destroy.

## Description

Terraform is configured as described in [common terraform setup](common-terraform-setup), followed by commands
equivalent to:

```shell-session
$ cd infra
$ terraform plan -destroy \
    -var-file=/build/release-metadata.json
$ 
$ terraform destroy -auto-approve \
    -var-file=/build/release-metadata.json
```
