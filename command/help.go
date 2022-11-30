package command

const globalOptions = `Global options:

  --component COMPONENT_NAME   - override component name (inferred from git by default).
  --commit GIT_COMMIT          - override the git commit (inferred from git by default).
  --no-pull-config             - don't pull the config container (must exist).
  --no-pull-release            - don't pull the release container (must exist).
  --no-pull-terraform          - don't pull the terraform container (must exist).
  --quiet | -q                 - hide verbose description of what's going on.
  --version                    - print the version number and exit. 
  --help                       - print the help message and exit.
`

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

  --name                            - Name of the new project repository
  --boilerplate                     - git URL of the git repo to copy as boilerplate. To use a specific branch (or any valid git refspec), add "?ref=branch-name" to the end of the URL.
  --{boilerplate arguments}         - Dynamic arguments for the templates files. E.g.: "--domain name --account test"

Boilerplate Templates:

    The boilerplate might include variable placeholders in any file in the repo
    with the format: %{name}

	The 'name' variable is predefined, using the value passed by --name.

    You can specify additional variables by passing arguments like:
        --domain name"
        --account test"

` + globalOptions

const globalOptionErrorFormat = `
Error in global options:

	%v

For usage run:

	cdflow --help

`
