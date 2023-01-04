package init

import (
	"bufio"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"strings"

	"github.com/mergermarket/cdflow2/command"
)

const (
	dirPermission  = fs.FileMode(0756)
	filePermission = fs.FileMode(0644)
)

// CommandArgs contains specific arguments to the init command.
type CommandArgs struct {
	Name        string
	Boilerplate string
	Variables   map[string]string
}

// RunCommand runs the init command.
func RunCommand(state *command.GlobalState, args *CommandArgs, env map[string]string) error {
	if args.Boilerplate != "" {
		return initFromBoilerplate(state, args)
	}

	return initFromBasicTemplate(state, args)
}

// ParseArgs parse command line arguments for init command.
func ParseArgs(args []string) (*CommandArgs, error) {
	if len(args)%2 != 0 {
		return nil, fmt.Errorf("argument length must be even, boolean arguments are not supported")
	}

	result := CommandArgs{Variables: map[string]string{}}
	i := 0

	take := func() (string, error) {
		i++
		if i >= len(args) {
			return "", errors.New("missing value")
		}

		return args[i], nil
	}

	for ; i < len(args); i++ {
		_, err := handleArgs(args[i], &result, take)
		if err != nil {
			return nil, err
		}
	}

	if result.Name == "" {
		return nil, errors.New("name argument is missing")
	}

	return &result, nil
}

func checkFolder(state *command.GlobalState, folder string) (bool, error) {
	info, err := os.Stat(folder)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}

	if !info.IsDir() {
		fmt.Fprintf(state.ErrorStream, "'%s' exists but not a folder\n", folder)
		fmt.Fprintf(state.ErrorStream, "Do you want to delete it? (y/N) ")

		reader := bufio.NewReader(state.InputStream)
		answer, _, err := reader.ReadRune()
		if err != nil {
			return false, err
		}

		switch answer {
		case 'y', 'Y':
			err := os.Remove(folder)
			if err != nil {
				return false, err
			}
		default:
			fmt.Fprintf(state.ErrorStream, "Delete the file '%s' or try another name.\n", folder)
			return false, fmt.Errorf("'%s' is not a folder", folder)
		}

		return false, nil
	}

	entries, err := os.ReadDir(folder)
	if err != nil {
		return false, err
	}

	if len(entries) == 0 {
		return true, nil
	}

	fmt.Fprintf(state.ErrorStream, "'%s' folder exist but not empty\n", folder)
	fmt.Fprintf(state.ErrorStream, "Do you want to delete all files and folders in '%s'? (y/N) ", folder)

	reader := bufio.NewReader(state.InputStream)
	answer, _, err := reader.ReadRune()
	if err != nil {
		return false, err
	}

	switch answer {
	case 'y', 'Y':
		err := os.RemoveAll(folder)
		if err != nil {
			return false, err
		}
	default:
		fmt.Fprintf(state.ErrorStream, "Delete the folder '%s' or try another name.\n", folder)
		return false, fmt.Errorf("'%s' is not empty", folder)
	}

	return false, nil
}

func handleArgs(arg string, commandArgs *CommandArgs, take func() (string, error)) (bool, error) {
	switch arg {
	case "--name":
		value, err := take()
		if err != nil {
			return false, err
		}
		commandArgs.Name = value
	case "--boilerplate":
		value, err := take()
		if err != nil {
			return false, err
		}
		commandArgs.Boilerplate = value
	default:
		if !strings.HasPrefix(arg, "--") {
			return false, fmt.Errorf("argument name must start with '--': %s", arg)
		}

		name := strings.TrimPrefix(arg, "--")

		value, err := take()
		if err != nil {
			return false, err
		}

		commandArgs.Variables[name] = value
	}

	return false, nil
}
