package main

import (
	"fmt"
	"math/rand"
	"os"
	"time"

	"github.com/mergermarket/cdflow2/command"
	"github.com/mergermarket/cdflow2/deploy"
	release "github.com/mergermarket/cdflow2/release/command"
)

var version = "undefined"

const globalArgs string = `Global args:

  --component COMPONENT_NAME   - override component name (inferred from git by default).
  --no-pull-config             - don't pull the config container (must exist).
  --no-pull-release            - don't pull the release container (must exist).
  --no-pull-terraform          - don't pull the terraform container (must exist).
  --version                    - print the version number and exit. 
  --help                       - print the help message and exit.
`

const help string = `
Usage:

  cdflow2 [ GLOBALARGS ] COMMAND [ ARGS ]

Commands:

  release VERSION       - build and publish a new software artefact
  deploy ENV VERSION    - create & update infrastructure using software artefact
  help [ COMMAND ]      - displayed detailed help and usage information for a command

` + globalArgs

const releaseHelp string = `
Usage:

  cdflow2 [ GLOBALARGS ] release VERSION

Args:

  VERSION     - the version being released. We recommend using evergreen version numbers (i.e. simple incrementing integers,
			    probably from your CI service), combined with something to identify the commit - e.g. "34-a5dbc4a7".

` + globalArgs

const deployHelp string = `
Usage:

  cdflow2 [ GLOBALARGS ] deploy ENV VERSION

Args:

  ENV         - the environment being deployed to.
  VERSION     - the version being deployed (must match what was released).

` + globalArgs

func usage() {
	fmt.Println(help)
	os.Exit(1)
}

func main() {
	rand.Seed(time.Now().UnixNano())

	globalArgs, remainingArgs, err := command.ParseArgs(os.Args[1:])

	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		usage()
	}
	if globalArgs.Command == "" || globalArgs.Command == "--help" {
		usage()
	} else if globalArgs.Command == "--version" {
		fmt.Println(version)
		os.Exit(0)
	}

	state, err := command.GetGlobalState(globalArgs)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	if globalArgs.Command == "release" {
		if len(remainingArgs) != 1 {
			fmt.Println(releaseHelp)
			os.Exit(1)
		}
		if err := release.RunCommand(state, remainingArgs[0]); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	} else if globalArgs.Command == "deploy" {
		if len(remainingArgs) != 2 {
			fmt.Println(deployHelp)
			os.Exit(1)
		}
		if err := deploy.RunCommand(state, remainingArgs[0], remainingArgs[1]); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	} else {
		usage()
	}
}
