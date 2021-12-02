---
name: Usage
menu: Commands
route: /commands/usage
---

# Usage

`cdflow2` is invoked with optional global arguments followed by a subcommand,
and then arguments specific to that subcommand:

```
cdflow2 [ GLOBALARGS ] COMMAND [ ARGS ]
```

## Commands

* [`setup`](setup) - interactive setup for your project.
* [`release`](release) - build and publish a release for a later deploymment.
* [`deploy`](deploy) - apply a release to an environment using Terraform.

## Global Options

<dl>
  <dt>`--component COMPONENT_NAME`</dt>
  <dd>Override component name (inferred from git by default).</dd>
  <dt>`--commit GIT_COMMIT`</dt>
  <dd>Override the git commit (inferred from git by default).</dd>
  <dt>`--no-pull-config`</dt>
  <dd>Don't pull the config container (must exist).</dd>
  <dt>`--no-pull-release`</dt>
  <dd>Don't pull the release container (must exist).</dd>
  <dt>`--no-pull-terraform`</dt>
  <dd>Don't pull the terraform container (must exist).</dd>
  <dt>`--quiet` \| `-q`</dt>
  <dd>Hide verbose description of what's going on.</dd>
  <dt>`--version`</dt>
  <dd>Print the version number and exit.</dd>
  <dt>`--help`</dt>
  <dd>Print the help message and exit.</dd>
</dl>
