# Destroy

## Usage

`cdflow2 [ GLOBALOPTS ] destroy [ OPTS ] ENV VERSION`

See [usage](./usage) for global options.

### Arguments:

<dl>
  <dt><code>ENV</code></dt>
  <dd>The environment containing the infrastructure being destroyed.</dd>
  <dt>VERSION</dt>
  <dd>The version to destroy (must match a pre-existing release).</dd>
</dl>

### Options:

<dl>
  <dt><code>--plan-only</code> | <code>-p</code></dt>
  <dd>Generate an execution plan only, don't destroy.</dd>
</dl>

## Description

Terraform is configured as described in [common terraform setup](common-terraform-setup), followed by commands
equivalent to:

```none
cd infra

terraform plan -destroy \
    -var-file=/build/release-metadata.json

terraform destroy -auto-approve \
    -var-file=/build/release-metadata.json
```
