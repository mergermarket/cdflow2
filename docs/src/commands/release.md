---
name: Release
menu: Commands
route: /release
---

# Release

## Usage

```
cdflow2 [ GLOBALARGS ] release VERSION

Args:

  VERSION     - the version being released. We recommend using evergreen version numbers (i.e. simple
                incrementing integers, probably from your CI service), combined with something to identify 
                the commit - e.g. "34-a5dbc4a7".

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

Release builds each of the `builds` configured in [`cdflow.yaml`](cdflow-yaml-reference#builds-optional),
as well as saving the terraform image and downloaded terraform modules and providrers against the provided
version number. This ensures that exactly what is deployed to one environment is the same as that promoted
to another.

The terraform command performed is equivalnet to:

```shell
terraform init -backend=false infra
```