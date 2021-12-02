# Destroy

## Usage

```
cdflow2 [ GLOBALOPTS ] destroy [ OPTS ] ENV VERSION

Args:

  ENV                 - the environment containing the infrastructure being destroyed.
  VERSION             - the version to destroy (must match a pre-existing release).

Options:

  --plan-only | -p    - generate an execution plan only, don't destroy.
```

## Description

Terraform is configured as described in [common terraform setup](common-terraform-setup), followed by commands
equivalent to:

```shell
cd infra

terraform plan -destroy \
    -var-file=/build/release-metadata.json

terraform destroy -auto-approve \
    -var-file=/build/release-metadata.json
```

## Options

```
--plan-only : 
Perform the terraform plan command only. The terraform destroy command is skipped.
```
