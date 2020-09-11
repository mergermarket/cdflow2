package main

import (
	"fmt"
	"os"

	"github.com/mergermarket/cdflow2/command"
	"github.com/mergermarket/cdflow2/deploy"
	"github.com/mergermarket/cdflow2/destroy"
	release "github.com/mergermarket/cdflow2/release/command"
	"github.com/mergermarket/cdflow2/setup"
	"github.com/mergermarket/cdflow2/shell"
	"github.com/mergermarket/cdflow2/util"
)

var version = "undefined"

const globalOptions string = `Global options:

  --component COMPONENT_NAME   - override component name (inferred from git by default).
  --commit GIT_COMMIT          - override the git commit (inferred from git by default).
  --no-pull-config             - don't pull the config container (must exist).
  --no-pull-release            - don't pull the release container (must exist).
  --no-pull-terraform          - don't pull the terraform container (must exist).
  --quiet | -q                 - hide verbose description of what's going on.
  --version                    - print the version number and exit. 
  --help                       - print the help message and exit.
`

const help string = `
Usage:

  cdflow2 [ GLOBALOPTS ] COMMAND [ ARGS ]

Commands:

  setup                   - configure your pipeline
  release VERSION         - build and publish a new software artefact
  deploy ENV VERSION      - create & update infrastructure using software artefact
  shell ENV VERSION       - access terraform for debugging and tf state manipulation
  destroy ENV [ VERSION ] - destroy all Terraform managed infrastructure in ENV
  help [ COMMAND ]        - display detailed help and usage information for a command

` + globalOptions

const releaseHelp string = `
Usage:

  cdflow2 [ GLOBALOPTS ] release [ OPTS ] VERSION

Args:

  VERSION                - the version being released. We recommend using evergreen version numbers (i.e. simple incrementing integers,
                           probably from your CI service), combined with something to identify the commit - e.g. "34-a5dbc4a7".

Options:

  --release-data | -r    - add key/value to release metadata (i.e. --release-data foo=bar).

` + globalOptions

const deployHelp string = `
Usage:

  cdflow2 [ GLOBALOPTS ] deploy [ OPTS ] ENV VERSION

Args:

  ENV         - the environment being deployed to.
  VERSION     - the version being deployed (must match what was released).

Options:

  --plan-only | -p    - create the terraform plan only, don't apply.

` + globalOptions

const setupHelp string = `
Usage:

  cdflow2 [ GLOBALARGS ] setup

` + globalOptions

const shellHelp string = `
Usage:

  cdflow2 [ GLOBALOPTS ] shell ENV [ OPTS ] [ SHELLARGS ]

Args:

  ENV         		- the environment containing the deployment.

Options:

  -v, --version     - the version to interract with (must match a pre-existing release).

Shell Arguments:

  The shell arguments are passed to shell 
  ex:  (cdflow2 shell aslive test.sh)
  	   (cdflow2 shell aslive -v v1.0 -- -c "echo test") or
`

const destroyHelp string = `
Usage:

  cdflow2 [ GLOBALOPTS ] destroy [ OPTS ] ENV [ VERSION ]

Args:

  ENV         - the environment containing the infrastructure being destroyed.
  VERSION     - the version to destroy (must match a pre-existing release).

Options:

  --plan-only | -p    - generate an execution plan only, don't destroy.

` + globalOptions

func usage(subcommand string) {
	if subcommand == "release" {
		fmt.Println(releaseHelp)
	} else if subcommand == "deploy" {
		fmt.Println(deployHelp)
	} else if subcommand == "shell" {
		fmt.Println(shellHelp)
	} else if subcommand == "setup" {
		fmt.Println(setupHelp)
	} else if subcommand == "destroy" {
		fmt.Println(destroyHelp)
	} else {
		fmt.Println(help)
	}
	os.Exit(1)
}

var globalOptionErrorFormat = `
Error in global options:

	%v

For usage run:

	cdflow --help

`

func main() {
	globalArgs, remainingArgs, err := command.ParseArgs(os.Args[1:])

	if err != nil {
		fmt.Fprintf(os.Stderr, globalOptionErrorFormat, err)
		os.Exit(1)
	}
	if globalArgs.Command == "" {
		usage("")
	} else if globalArgs.Command == "help" {
		subcommand := ""
		if len(remainingArgs) > 0 {
			subcommand = remainingArgs[0]
		}
		usage(subcommand)
	} else if globalArgs.Command == "version" {
		fmt.Println(version)
		os.Exit(0)
	}

	state, err := command.GetGlobalState(globalArgs)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	env := util.GetEnv(os.Environ())

	if globalArgs.Command == "release" {
		releaseArgs, ok := release.ParseArgs(remainingArgs)
		if ok != nil {
			fmt.Fprintln(os.Stderr, fmt.Sprintf("Error: %s", ok))
			usage("release")
		}
		if err := release.RunCommand(state, *releaseArgs, env); err != nil {
			if status, ok := err.(command.Failure); ok {
				os.Exit(int(status))
			}
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	} else if globalArgs.Command == "deploy" {
		deployArgs, ok := deploy.ParseArgs(remainingArgs)
		if !ok {
			usage("deploy")
		}
		if err := deploy.RunCommand(state, deployArgs, env); err != nil {
			if status, ok := err.(command.Failure); ok {
				os.Exit(int(status))
			}
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	} else if globalArgs.Command == "shell" {
		shellArgs, ok := shell.ParseArgs(remainingArgs)
		if ok != nil {
			usage("shell")
		}
		if err := shell.RunCommand(state, shellArgs, env); err != nil {
			if status, ok := err.(command.Failure); ok {
				os.Exit(int(status))
			}
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

	} else if globalArgs.Command == "setup" {
		if len(remainingArgs) != 0 {
			usage("setup")
		}
		if err := setup.RunCommand(state, env); err != nil {
			if status, ok := err.(command.Failure); ok {
				os.Exit(int(status))
			}
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	} else if globalArgs.Command == "destroy" {
		destroyArgs, ok := destroy.ParseArgs(remainingArgs)
		if !ok {
			usage("destroy")
		}
		if err := destroy.RunCommand(state, destroyArgs, env); err != nil {
			if status, ok := err.(command.Failure); ok {
				os.Exit(int(status))
			}
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	} else {
		usage("")
	}
}
