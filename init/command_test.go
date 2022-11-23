package init

import (
	"bytes"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mergermarket/cdflow2/command"
)

func TestParseArgs(t *testing.T) {
	commandArgs, err := ParseArgs([]string{"--name", "test-name", "--boilerplate", "boilerplate-url", "--init-1", "value-1", "--init-2", "value-2"})
	if err != nil {
		t.Fatal("unexpected error from parseArgs:", err)
	}

	if commandArgs.Boilerplate != "boilerplate-url" {
		t.Error("expecting boilerplate-url name, got: ", commandArgs.Boilerplate)
	}

	if commandArgs.Name != "test-name" {
		t.Error("expecting test-name name, got: ", commandArgs.Name)
	}

	if len(commandArgs.Variables) != 2 {
		t.Error("expecting variable len 2, got: ", len(commandArgs.Variables))
	}

	if commandArgs.Variables["init-1"] != "value-1" || commandArgs.Variables["init-2"] != "value-2" {
		t.Error("expecting variables to have 'init-1=value-1' and 'init-2=value-2', got: ", commandArgs.Variables)
	}
}

func TestParseArgsInvalid(t *testing.T) {
	t.Run("boilerplate", func(t *testing.T) {
		commandArgs, err := ParseArgs([]string{"--boilerplate", "boilerplate-url"})
		if err != nil {
			t.Fatal("unexpected error from parseArgs:", err)
		}

		err = RunCommand(&command.GlobalState{ErrorStream: os.Stderr, CodeDir: t.TempDir()}, commandArgs, nil)

		if err == nil {
			t.Error("expected error from RunCommand, but not getting it")
		}
	})

	t.Run("basic", func(t *testing.T) {
		commandArgs, err := ParseArgs([]string{})
		if err != nil {
			t.Fatal("unexpected error from parseArgs:", err)
		}

		err = RunCommand(&command.GlobalState{ErrorStream: os.Stderr, CodeDir: t.TempDir()}, commandArgs, nil)

		if err == nil {
			t.Error("expected error from RunCommand, but not getting it")
		}
	})
}

func TestBasicTemplate(t *testing.T) {
	_ = os.Setenv("GIT_AUTHOR_NAME", "cdflow2")
	_ = os.Setenv("GIT_AUTHOR_EMAIL", "cdflow2")
	_ = os.Setenv("GIT_COMMITTER_NAME", "cdflow2")
	_ = os.Setenv("GIT_COMMITTER_EMAIL", "cdflow2")

	assertFiles := func(tmpDir string, name string) {
		baseDir := filepath.Join(tmpDir, name)

		expectedFiles := []string{
			filepath.Join(baseDir, "cdflow.yaml"),
			filepath.Join(baseDir, "config", "common.json"),
			filepath.Join(baseDir, "infra", "main.tf"),
			filepath.Join(baseDir, "infra", "output.tf"),
			filepath.Join(baseDir, "infra", "variables.tf"),
			filepath.Join(baseDir, "infra", "version.tf"),
		}

		for _, file := range expectedFiles {
			_, err := os.Stat(file)
			if err != nil {
				if os.IsNotExist(err) {
					t.Errorf("expected file does not exist: %s", file)
				}

				t.Fatal(err)
			}
		}
	}

	t.Run("repo not exist", func(t *testing.T) {
		tmpDir := t.TempDir()
		name := "basic-test"

		state := &command.GlobalState{ErrorStream: os.Stderr, CodeDir: tmpDir}
		commandArgs := &CommandArgs{Name: name}

		err := RunCommand(state, commandArgs, nil)
		if err != nil {
			t.Fatalf("unexpected error from RunCommand: %v", err)
		}

		assertFiles(tmpDir, name)
	})

	t.Run("repo exist but empty", func(t *testing.T) {
		tmpDir := t.TempDir()
		name := "basic-test"

		err := os.Mkdir(filepath.Join(tmpDir, name), 0765)
		if err != nil {
			t.Fatal(err)
		}

		state := &command.GlobalState{ErrorStream: os.Stderr, CodeDir: tmpDir}
		commandArgs := &CommandArgs{Name: name}

		err = RunCommand(state, commandArgs, nil)
		if err != nil {
			t.Fatalf("unexpected error from RunCommand: %v", err)
		}

		assertFiles(tmpDir, name)
	})

	t.Run("repo is a file - delete", func(t *testing.T) {
		tmpDir := t.TempDir()
		name := "basic-test"

		err := os.WriteFile(filepath.Join(tmpDir, name), nil, 0644)
		if err != nil {
			t.Fatal(err)
		}

		stdin := strings.NewReader("y")

		state := &command.GlobalState{ErrorStream: os.Stderr, InputStream: stdin, CodeDir: tmpDir}
		commandArgs := &CommandArgs{Name: name}

		err = RunCommand(state, commandArgs, nil)
		if err != nil {
			t.Fatalf("unexpected error from RunCommand: %v", err)
		}

		assertFiles(tmpDir, name)
	})

	t.Run("repo is a not empty folder - delete", func(t *testing.T) {
		tmpDir := t.TempDir()
		name := "basic-test"

		err := os.Mkdir(filepath.Join(tmpDir, name), 0765)
		if err != nil {
			t.Fatal(err)
		}

		err = os.WriteFile(filepath.Join(tmpDir, name, "test"), nil, 0644)
		if err != nil {
			t.Fatal(err)
		}

		stdin := strings.NewReader("y")

		state := &command.GlobalState{ErrorStream: os.Stderr, InputStream: stdin, CodeDir: tmpDir}
		commandArgs := &CommandArgs{Name: name}

		err = RunCommand(state, commandArgs, nil)
		if err != nil {
			t.Fatalf("unexpected error from RunCommand: %v", err)
		}

		assertFiles(tmpDir, name)
	})

	t.Run("repo is a file - not delete", func(t *testing.T) {
		tmpDir := t.TempDir()
		name := "basic-test"

		err := os.WriteFile(filepath.Join(tmpDir, name), nil, 0644)
		if err != nil {
			t.Fatal(err)
		}

		stdin := strings.NewReader("n")

		state := &command.GlobalState{ErrorStream: os.Stderr, InputStream: stdin, CodeDir: tmpDir}
		commandArgs := &CommandArgs{Name: name}

		err = RunCommand(state, commandArgs, nil)
		if err == nil {
			t.Fatalf("expected error from RunCommand")
		}
	})

	t.Run("repo is a not empty folder - not delete", func(t *testing.T) {
		tmpDir := t.TempDir()
		name := "basic-test"

		err := os.Mkdir(filepath.Join(tmpDir, name), 0765)
		if err != nil {
			t.Fatal(err)
		}

		err = os.WriteFile(filepath.Join(tmpDir, name, "test"), nil, 0644)
		if err != nil {
			t.Fatal(err)
		}

		stdin := strings.NewReader("n")

		state := &command.GlobalState{ErrorStream: os.Stderr, InputStream: stdin, CodeDir: tmpDir}
		commandArgs := &CommandArgs{Name: name}

		err = RunCommand(state, commandArgs, nil)
		if err == nil {
			t.Fatalf("expected error from RunCommand")
		}
	})
}

func TestBoilerplate(t *testing.T) {
	_ = os.Setenv("GIT_AUTHOR_NAME", "cdflow2")
	_ = os.Setenv("GIT_AUTHOR_EMAIL", "cdflow2")
	_ = os.Setenv("GIT_COMMITTER_NAME", "cdflow2")
	_ = os.Setenv("GIT_COMMITTER_EMAIL", "cdflow2")

	gitRepoPath := t.TempDir()
	err := createTestRepo(gitRepoPath)
	if err != nil {
		t.Fatalf("unable to create test git repository: %v", err)
	}

	t.Run("missing variable", func(t *testing.T) {
		tmpDir := t.TempDir()
		name := "boilerplate"

		state := &command.GlobalState{ErrorStream: os.Stderr, CodeDir: tmpDir}
		commandArgs := &CommandArgs{
			Name:        name,
			Boilerplate: gitRepoPath,
			Variables:   nil,
		}

		err = RunCommand(state, commandArgs, nil)
		if err == nil {
			t.Fatalf("expected error from RunCommand")
		}
	})

	t.Run("rendering successful", func(t *testing.T) {
		wd, err := os.Getwd()
		if err != nil {
			t.Fatal(err)
		}

		tmpDir := t.TempDir()
		name := "boilerplate"

		state := &command.GlobalState{ErrorStream: os.Stderr, CodeDir: tmpDir}
		commandArgs := &CommandArgs{
			Name:        name,
			Boilerplate: gitRepoPath,
			Variables:   map[string]string{"env": "test", "org": "ION", "team": "platform"},
		}

		err = RunCommand(state, commandArgs, nil)
		if err != nil {
			t.Fatalf("unexpected error from RunCommand: %v", err)
		}

		expectedBasePath := filepath.Join(wd, "testdata", "expected")

		err = filepath.WalkDir(expectedBasePath, func(path string, dirEntry fs.DirEntry, err error) error {
			if err != nil {
				return err
			}

			if dirEntry.IsDir() {
				return nil
			}

			relativePath, err := filepath.Rel(expectedBasePath, path)
			if err != nil {
				t.Fatal(err)
			}

			expected, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}

			actual, err := os.ReadFile(filepath.Join(tmpDir, name, relativePath))
			if err != nil {
				t.Fatal(err)
			}

			if !bytes.Equal(expected, actual) {
				t.Fatalf("rendered template mismatch, expected: %s\nactual: %s", string(expected), string(actual))
			}

			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	})
}

func createTestRepo(tmpDir string) error {
	rootPath := filepath.Join("testdata", "repo")

	err := filepath.WalkDir(rootPath, func(path string, dirEntry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(rootPath, path)
		if err != nil {
			return err
		}

		if dirEntry.IsDir() {
			err := os.MkdirAll(filepath.Join(tmpDir, relPath), 0765)
			if err != nil {
				return err
			}
		} else {
			b, err := os.ReadFile(path)
			if err != nil {
				return err
			}

			err = os.WriteFile(filepath.Join(tmpDir, relPath), b, 0644)
			if err != nil {
				return err
			}
		}

		return nil
	})
	if err != nil {
		return err
	}

	cmd := exec.Command("git", "init")
	cmd.Dir = tmpDir
	err = cmd.Run()
	if err != nil {
		return err
	}

	cmd = exec.Command("git", "add", ".")
	cmd.Dir = tmpDir
	err = cmd.Run()
	if err != nil {
		return err
	}

	cmd = exec.Command("git", "commit", "-m", "Create repo for test")
	cmd.Dir = tmpDir
	err = cmd.Run()
	if err != nil {
		return err
	}

	return nil
}
