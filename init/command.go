package init

import (
	"bufio"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
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
	Team        string
	Org         string
	InitVars    map[string]string
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
	result := CommandArgs{InitVars: map[string]string{}}
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

func initRepo(state *command.GlobalState, folder string) error {
	fmt.Fprintf(state.ErrorStream, "Init git repository.\n")

	cmd := exec.Command("git", "init", "-b", "main")
	cmd.Dir = folder

	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Fprintf(state.ErrorStream, string(output))
		return err
	}

	return nil
}

func commitChanges(state *command.GlobalState, folder string) error {
	fmt.Fprintf(state.ErrorStream, "Commit changes.\n")

	cmd := exec.Command("git", "add", ".")
	cmd.Dir = folder

	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Fprintf(state.ErrorStream, string(output))
		return err
	}

	cmd = exec.Command("git", "commit", "-m", "Create initial project.")
	cmd.Dir = folder

	output, err = cmd.CombinedOutput()
	if err != nil {
		fmt.Fprintf(state.ErrorStream, string(output))
		return err
	}

	return nil
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
	case "--team":
		value, err := take()
		if err != nil {
			return false, err
		}
		commandArgs.Team = value
	case "--org":
		value, err := take()
		if err != nil {
			return false, err
		}
		commandArgs.Org = value
	case "--init-var":
		value, err := take()
		if err != nil {
			return false, err
		}

		parts := strings.Split(value, "=")
		if len(parts) != 2 {
			return false, fmt.Errorf("invalid argument for 'init-var': %s", value)
		}

		if parts[0] == "" || parts[1] == "" {
			return false, fmt.Errorf("init-var key and value must not be empty: %s", value)
		}

		commandArgs.InitVars[parts[0]] = parts[1]
	default:
		return false, fmt.Errorf("unknown option: %s", arg)
	}

	return false, nil
}
