---
name: Release
menu: Commands
route: /commands/release
---

# Release

## Usage

`cdflow2 [ GLOBALARGS ] release VERSION`

See [usage](./usage) for global options.

### Arguments:

`VERSION`
: The version being released. We recommend using evergreen version numbers (i.e. simple incrementing integers, probably from your CI service), combined with something to identify the commit - e.g. "34-a5dbc4a7".

## Description

Release builds each of the `builds` configured in [`cdflow.yaml`](../cdflow-yaml-reference#builds-optional),
as well as saving the terraform image and downloaded terraform modules and providrers against the provided
version number. This ensures that exactly what is deployed to one environment is the same as that promoted
to another.

The terraform command performed is equivalent to:

```shell-session
$ cd infra
$ terraform init -backend=false
```
