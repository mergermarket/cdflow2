---
name: Shell
menu: Commands
route: /commands/shell
---

# Shell

## Usage

`cdflow2 [ GLOBALOPTS ] shell [ OPTS ] ENV -- SHELLARGS`

See [usage](./usage) for global options.

### Arguments:

`ENV`
: The environment being deployed to.

### Options

`--version` | `-v`
: The released version to use to setup terraform (currently an option, but may not work without - may be made a required parameter).

## Description

Terraform is configured as described in [common terraform setup](common-terraform-setup), followed by creating a shell.

The shell may be used interactively:

```shell-session
$ cdflow2 shell --version my-version live
# terraform ...
```

Or may be used to run a script (demonstrating SHELLARGS to pass arguments to the shell):

```shell-session
$ cdflow2 shell --version my-version live -- my-script.sh
```

As you would expect, reading piping in the script should also work:

```shell-session
$ echo 'terraform -version' | cdflow2 shell --version my-version live
```
