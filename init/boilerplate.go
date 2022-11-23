package init

import (
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/mergermarket/cdflow2/command"
)

var placeHolderPattern = regexp.MustCompile(`%\{(.+?)}`)

func initFromBoilerplate(state *command.GlobalState, args *CommandArgs) error {
	err := validateArgs(state, args)
	if err != nil {
		return err
	}

	projectFolder := filepath.Join(state.CodeDir, args.Name)

	found, err := checkFolder(state, projectFolder)
	if err != nil {
		return err
	}

	if !found {
		fmt.Fprintf(state.ErrorStream, "Create folder '%s'.\n", projectFolder)

		err := os.Mkdir(projectFolder, dirPermission)
		if err != nil {
			return err
		}
	}

	err = downloadBoilerplate(state, args.Boilerplate, projectFolder)
	if err != nil {
		return err
	}

	err = renderTemplate(state, args, projectFolder)
	if err != nil {
		return err
	}

	return nil
}

func downloadBoilerplate(state *command.GlobalState, url string, folder string) error {
	fmt.Fprintf(state.ErrorStream, "Downloading boilerplate from '%s' to '%s'\n", url, folder)

	cmdArgs := []string{"clone", "--depth", "1"}

	repository, branch, found := strings.Cut(url, "?ref=")
	if found {
		cmdArgs = append(cmdArgs, "--branch", branch)
	}

	cmdArgs = append(cmdArgs, repository, folder)

	cmd := exec.Command("git", cmdArgs...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Fprintf(state.ErrorStream, string(output))
		return err
	}

	err = os.RemoveAll(filepath.Join(folder, ".git"))
	if err != nil {
		return err
	}

	return nil
}

func renderTemplate(state *command.GlobalState, args *CommandArgs, folder string) error {
	fmt.Fprintf(state.ErrorStream, "Rendering templates...\n")
	defer func() { fmt.Fprintf(state.ErrorStream, "Rendering finished.\n") }()

	variables := map[string]string{
		"name": args.Name,
	}

	for k, v := range args.Variables {
		variables[k] = v
	}

	missingVariables := make(map[string]struct{})

	err := filepath.WalkDir(folder, func(path string, dirEntry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if dirEntry.IsDir() {
			if filepath.Base(path) == ".git" {
				return fs.SkipDir
			}
			return nil
		}

		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		content := string(b)

		replacedPlaceholders := make(map[string]struct{})

		matches := placeHolderPattern.FindAllStringSubmatch(content, -1)
		for _, match := range matches {
			for _, placeholder := range match[1:] {
				value, ok := variables[placeholder]
				if !ok {
					missingVariables[placeholder] = struct{}{}
					continue
				}

				if _, ok := replacedPlaceholders[placeholder]; ok {
					continue
				}

				content = strings.ReplaceAll(content, fmt.Sprintf("%%{%s}", placeholder), value)
				replacedPlaceholders[placeholder] = struct{}{}
			}
		}

		err = os.WriteFile(path, []byte(content), dirEntry.Type())
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return err
	}

	if len(missingVariables) != 0 {
		for variable := range missingVariables {
			fmt.Fprintf(state.ErrorStream, "\nVariable '%s' not defined.\n", variable)
			fmt.Fprintf(state.ErrorStream, "To add the missing variable:\n\n")
			fmt.Fprintf(state.ErrorStream, "--%s {value}\n", variable)
		}

		fmt.Fprintf(state.ErrorStream, "\nFor a detailed explanation of the missing variable(s) please see boilerplate README.md\n")

		return fmt.Errorf("required variables are missing")
	}

	return nil
}

func validateArgs(state *command.GlobalState, args *CommandArgs) error {
	invalid := false

	if args.Name == "" {
		fmt.Fprintf(state.ErrorStream, "'name' argument is empty\n")
		invalid = true
	}

	if args.Boilerplate == "" {
		fmt.Fprintf(state.ErrorStream, "'boilerplate' argument is empty\n")
		invalid = true
	}

	if invalid {
		return fmt.Errorf("some arguments are missing")
	}

	return nil
}
