# Deploy

## Usage

<code>cdflow2 [ GLOBALOPTS ] deploy [ OPTS ] ENV VERSION</code>

See [usage](./usage) for global options.

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

### First deployment to an environment

The `--new-state` or `-n` flag is required for the first deployment into a particular
environment, but then must be removed for subsequent deployments. This is a safety
feature that ensures you do not lose track of your tfstate (e.g. if the component name
changes but you haven't moved the statefile accordingly).