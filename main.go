package main

import (
	"fmt"
	"math/rand"
	"os"
	"time"

	"github.com/mergermarket/cdflow2/command"
	"github.com/mergermarket/cdflow2/deploy"
	"github.com/mergermarket/cdflow2/release"
)

const globalArgs string = `Global args:

  --component COMPONENT_NAME   - override component name (inferred from git by default).
  --no-pull-config             - don't pull the config container (must exist).
  --no-pull-release            - don't pull the release container (must exist).
  --no-pull-terraform          - don't pull the terraform container (must exist).
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

func main() {
	rand.Seed(time.Now().UnixNano())

	cmd, remainingArgs, env := command.GetGlobalEnv()

	if cmd == "release" {
		if len(remainingArgs) != 1 {
			fmt.Println(releaseHelp)
			os.Exit(1)
		}
		if err := release.RunCommand(env, remainingArgs[0]); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	} else if cmd == "deploy" {
		if len(remainingArgs) != 2 {
			fmt.Println(deployHelp)
			os.Exit(1)
		}
		if err := deploy.RunCommand(env, remainingArgs[0], remainingArgs[1]); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	} else {
		fmt.Println(help)
		os.Exit(1)
	}
}
