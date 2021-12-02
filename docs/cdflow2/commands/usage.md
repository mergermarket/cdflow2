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
  <dt><code>--component COMPONENT_NAME</code></dt>
  <dd>Override component name (inferred from git by default).</dd>
  <dt><code>--commit GIT_COMMIT</code></dt>
  <dd>Override the git commit (inferred from git by default).</dd>
  <dt><code>--no-pull-config</code></dt>
  <dd>Don't pull the config container (must exist).</dd>
  <dt><code>--no-pull-release</code></dt>
  <dd>Don't pull the release container (must exist).</dd>
  <dt><code>--no-pull-terraform</code></dt>
  <dd>Don't pull the terraform container (must exist).</dd>
  <dt><code>--quiet</code> | <code>-q</code></dt>
  <dd>Hide verbose description of what's going on.</dd>
  <dt><code>--version</code></dt>
  <dd>Print the version number and exit.</dd>
  <dt><code>--help</code></dt>
  <dd>Print the help message and exit.</dd>
</dl>
