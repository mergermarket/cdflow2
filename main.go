package main

import (
	"fmt"
	"os"

	"github.com/mergermarket/cdflow2/command"
	"github.com/mergermarket/cdflow2/deploy"
	"github.com/mergermarket/cdflow2/destroy"
	cinit "github.com/mergermarket/cdflow2/init"
	release "github.com/mergermarket/cdflow2/release/command"
	"github.com/mergermarket/cdflow2/setup"
	"github.com/mergermarket/cdflow2/shell"
	"github.com/mergermarket/cdflow2/util"
)

var version = "undefined"

const globalOptions = `Global options:

  --component COMPONENT_NAME   - override component name (inferred from git by default).
  --commit GIT_COMMIT          - override the git commit (inferred from git by default).
  --no-pull-config             - don't pull the config container (must exist).
  --no-pull-release            - don't pull the release container (must exist).
  --no-pull-terraform          - don't pull the terraform container (must exist).
  --version                    - print the version number and exit. 
  --help                       - print the help message and exit.`

const help = `
Usage:

  cdflow2 [ GLOBALOPTS ] COMMAND [ ARGS ]

Commands:

  setup                                   - configure your pipeline
  init    [ OPTS ]                        - initialize a new project
  release [ OPTS ] VERSION                - build and publish a new software artifact
  deploy  [ OPTS ] ENV VERSION            - create & update infrastructure using software artifact
  destroy [ OPTS ] ENV VERSION            - destroy all Terraform managed infrastructure in ENV
  shell   ENV [ OPTS ] [ SHELLARGS ]      - access terraform for debugging and tf state manipulation
  help    [ COMMAND ]                     - display detailed help and usage information for a command

` + globalOptions

const releaseHelp = `
Usage:

  cdflow2 [ GLOBALOPTS ] release [ OPTS ] VERSION

Args:

  VERSION                - the version being released. We recommend using evergreen version numbers (i.e. simple incrementing integers,
                           probably from your CI service), combined with something to identify the commit - e.g. "34-a5dbc4a7".

Options:

  --release-data | -r    - add key/value to release metadata (i.e. --release-data foo=bar).

` + globalOptions

const deployHelp = `
Usage:

  cdflow2 [ GLOBALOPTS ] deploy [ OPTS ] ENV VERSION

Args:

  ENV                 - the environment being deployed to.
  VERSION             - the version being deployed (must match what was released).

Options:

  --plan-only | -p    - create the terraform plan only, don't apply.
  --new-state | -n    - allow run without a pre-existing tfstate file.

` + globalOptions

const setupHelp = `
Usage:

  cdflow2 [ GLOBALARGS ] setup

` + globalOptions

const shellHelp = `
Usage:

  cdflow2 [ GLOBALOPTS ] shell ENV [ OPTS ] [ SHELLARGS ]

Args:

  ENV               - the environment containing the deployment.

Options:

  -v, --version     - followed by the name of which version to interract with (must match a pre-existing release).

Shell Arguments:

  The shell arguments are passed to shell 
  ex:  (cdflow2 shell demo test.sh)
  	   (cdflow2 shell demo -v v1.0 -- -c "echo test")
` + globalOptions

const destroyHelp = `
Usage:

  cdflow2 [ GLOBALOPTS ] destroy [ OPTS ] ENV VERSION

Args:

  ENV                 - the environment containing the infrastructure being destroyed.
  VERSION             - the version to destroy (must match a pre-existing release).

Options:

  --plan-only | -p    - generate an execution plan only, don't destroy.

` + globalOptions

const initHelp = `
Usage:

  cdflow2 [ GLOBALOPTS ] init [ OPTS ]

Options:

  --name | -n                       - Name of the new project repository
  --boilerplate | -b                - git URL of the git repo to copy as boilerplate. To use a specific branch (or any valid git refspec), add "?ref=branch-name" to the end of the URL.
  --{boilerplate arguments}         - Dynamic arguments for the templates files. E.g.: "--domain name --account test"

Boilerplate Templates:

    The boilerplate might include variable placeholders in any file in the repo
    with the format: %{name}

	The 'name' variable is predefined, using the value passed by --name.

    You can specify additional variables by passing arguments like:
        --domain name
        --account test

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
	} else if subcommand == "init" {
		fmt.Println(initHelp)
	} else {
		fmt.Println(help)
	}
}

var globalOptionErrorFormat = `
Error in global options:

	%v

For usage run:

	cdflow --help

`

func main() {
	os.Exit(runCommand())
}

func runCommand() (status int) {
	globalArgs, remainingArgs, err := command.ParseArgs(os.Args[1:])

	if err != nil {
		fmt.Fprintf(os.Stderr, globalOptionErrorFormat, err)
		return 2
	}
	if globalArgs.Command == "" {
		usage("")
		return 2
	} else if globalArgs.Command == "help" {
		subcommand := ""
		if len(remainingArgs) > 0 {
			subcommand = remainingArgs[0]
		}
		usage(subcommand)
		return 0
	} else if globalArgs.Command == "version" {
		fmt.Println(version)
		return 0
	}

	state, err := command.GetGlobalState(globalArgs, globalArgs.Command != "init")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	defer func() {
		if globalArgs.Command == "init" || status == 2 {
			return
		}

		state.MonitoringClient.Command = globalArgs.Command
		state.MonitoringClient.Project = state.Component
		state.MonitoringClient.Version = version
		state.MonitoringClient.StatusCode = status

		state.MonitoringClient.SubmitEvent()
	}()

	env := util.GetEnv(os.Environ())

	if globalArgs.Command == "release" {
		releaseArgs, err := release.ParseArgs(remainingArgs)
		if err != nil {
			fmt.Fprintln(os.Stderr, fmt.Sprintf("Error: %s", err))
			usage("release")
			return 2
		}

		if err := release.RunCommand(state, *releaseArgs, env); err != nil {
			if status, ok := err.(command.Failure); ok {
				return int(status)
			}
			fmt.Fprintln(os.Stderr, "\n"+err.Error())
			return 1
		}
	} else if globalArgs.Command == "deploy" {
		deployArgs, err := deploy.ParseArgs(remainingArgs)
		if err != nil {
			fmt.Fprintln(os.Stderr, fmt.Sprintf("Error: %s", err))
			usage("deploy")
			return 2
		}

		state.MonitoringClient.Environment = deployArgs.EnvName

		if err := deploy.RunCommand(state, deployArgs, env); err != nil {
			if status, ok := err.(command.Failure); ok {
				return int(status)
			}
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
	} else if globalArgs.Command == "shell" {
		shellArgs, err := shell.ParseArgs(remainingArgs)
		if err != nil {
			fmt.Fprintln(os.Stderr, fmt.Sprintf("Error: %s", err))
			usage("shell")
			return 2
		}

		state.MonitoringClient.Environment = shellArgs.EnvName

		if err := shell.RunCommand(state, shellArgs, env); err != nil {
			if status, ok := err.(command.Failure); ok {
				return int(status)
			}
			fmt.Fprintln(os.Stderr, err)
			return 1
		}

	} else if globalArgs.Command == "setup" {
		if len(remainingArgs) != 0 {
			fmt.Fprintln(os.Stderr, "Error: setup has no arguments")
			usage("setup")
			return 2
		}

		if err := setup.RunCommand(state, env); err != nil {
			if status, ok := err.(command.Failure); ok {
				return int(status)
			}
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
	} else if globalArgs.Command == "destroy" {
		destroyArgs, err := destroy.ParseArgs(remainingArgs)
		if err != nil {
			fmt.Fprintln(os.Stderr, fmt.Sprintf("Error: %s", err))
			usage("destroy")
			return 2
		}

		state.MonitoringClient.Environment = destroyArgs.EnvName

		if err := destroy.RunCommand(state, destroyArgs, env); err != nil {
			if status, ok := err.(command.Failure); ok {
				return int(status)
			}
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
	} else if globalArgs.Command == "init" {
		initArgs, err := cinit.ParseArgs(remainingArgs)
		if err != nil {
			fmt.Fprintln(os.Stderr, fmt.Sprintf("Error: %s", err))
			usage("init")
			return 2
		}

		if err := cinit.RunCommand(state, initArgs, env); err != nil {
			if status, ok := err.(command.Failure); ok {
				return int(status)
			}
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
	} else {
		usage("")
		return 2
	}

	return 0
}
