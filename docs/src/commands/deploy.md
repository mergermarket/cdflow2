---
name: Deploy
menu: Commands
route: /deploy
---

# Deploy

## Usage

```
cdflow2 [ GLOBALARGS ] deploy ENV VERSION

Args:

  ENV         - the environment being deployed to.
  VERSION     - the version being deployed (must match what was released).

Global args:

  --component COMPONENT_NAME   - override component name (inferred from git by default).
  --commit GIT_COMMIT          - override the git commit (inferred from git by default).
  --no-pull-config             - don't pull the config container (must exist).
  --no-pull-release            - don't pull the release container (must exist).
  --no-pull-terraform          - don't pull the terraform container (must exist).
  --quiet | -q                 - hide verbose description of what's going on.
  --version                    - print the version number and exit.
  --help                       - print the help message and exit.
```

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
