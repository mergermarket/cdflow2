# Deploy

## Usage

<code>cdflow2 [ GLOBALOPTS ] deploy [ OPTS ] ENV VERSION</code>

### Arguments:

<dl>
  <dt>ENV</dt>
  <dd>The environment being deployed to.</dd>
  <dt>VERSION</dt>
  <dd>The version being deployed (must match what was released).</dd>
</dl>

### Options:

<dl>
  <dt>--plan-only | -p</dt>
  <dd>Create the terraform plan only, don't apply.</dd>
  <dt>--new-state | -n</dt>
  <dd>Allow run without a pre-existing tfstate file.</dd>
</dl>

## Description

Terraform is configured as described in [common terraform setup](common-terraform-setup), followed by commands
equivalent to:

```shell
terraform plan \
    -input=false \
    -var-file release-metadata-VERSION.json \
    -var-file config/ENV.json \
    -out=plan-TIMESTAMP \
    infra/

terraform apply \
    -input=false \
    plan-TIMESTAMP
```

## Options

```
--plan-only : 
Perform the terraform plan command only. The terraform apply command is skipped.

--new-state :
Toggles a parameter sent to the config container. 
It is received before terraform commands are performed and used to trigger state existance validations.
```