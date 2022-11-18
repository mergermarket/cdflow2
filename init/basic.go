package init

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/mergermarket/cdflow2/command"
)

//go:embed template
var basicTemplate embed.FS

func initFromBasicTemplate(state *command.GlobalState, args *CommandArgs) error {
	if args.Name == "" {
		return fmt.Errorf("'name' argument is empty")
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

	err = initRepo(state, projectFolder)
	if err != nil {
		return err
	}

	templateDir, err := fs.Sub(basicTemplate, "template")
	if err != nil {
		return err
	}

	fmt.Fprintf(state.ErrorStream, "Copy templates.\n")

	err = fs.WalkDir(templateDir, ".", func(path string, dirEntry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if path == "." {
			return nil
		}

		if dirEntry.IsDir() {
			err := os.Mkdir(filepath.Join(projectFolder, path), dirPermission)
			if err != nil {
				return err
			}
		} else {
			b, err := fs.ReadFile(templateDir, path)
			if err != nil {
				return err
			}

			err = os.WriteFile(filepath.Join(projectFolder, path), b, filePermission)
			if err != nil {
				return err
			}
		}

		return nil
	})
	if err != nil {
		return err
	}

	return commitChanges(state, projectFolder)
}
