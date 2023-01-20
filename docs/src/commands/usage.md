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
* [`destroy`](destroy) - destroy all resources in an environment.
* [`shell`](shell) - run a shell with Terraform configured.

## Global Options

`--component COMPONENT_NAME`
: Override component name (inferred from git by default).

`--commit GIT_COMMIT`
: Override the git commit (inferred from git by default).

`--no-pull-config`
: Don't pull the config container (must exist).

`--no-pull-release`
: Don't pull the release container (must exist).

`--no-pull-terraform`
: Don't pull the terraform container (must exist).

`--version`
: Print the version number and exit.

`--help`
: Print the help message and exit.
