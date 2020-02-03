package command_test

import (
	"log"
	"reflect"
	"testing"

	"github.com/mergermarket/cdflow2/command"
)

func TestParseArgs(t *testing.T) {
	globalArgs, remainingArgs := command.ParseArgs([]string{"test-command", "1", "2", "3"})
	if globalArgs.Command != "test-command" {
		log.Fatalln("expecting test-command command, got:", globalArgs.Command)
	}
	if !reflect.DeepEqual(remainingArgs, []string{"1", "2", "3"}) {
		log.Fatalln("unexpected remaining args:", remainingArgs)
	}
}
func TestParseArgsEmpty(t *testing.T) {
	globalArgs, remainingArgs := command.ParseArgs([]string{})
	if globalArgs.Command != "" {
		log.Fatalln("expecting empty test-command command, got:", globalArgs.Command)
	}
	if !reflect.DeepEqual(remainingArgs, []string{}) {
		log.Fatalln("unexpected empty remaining args, got:", remainingArgs)
	}
}

func TestParseArgsCommandOnly(t *testing.T) {
	globalArgs, remainingArgs := command.ParseArgs([]string{"test-command"})
	if globalArgs.Command != "test-command" {
		log.Fatalln("expecting empty test-command command, got:", globalArgs.Command)
	}
	if !reflect.DeepEqual(remainingArgs, []string{}) {
		log.Fatalln("unexpected empty remaining args, got:", remainingArgs)
	}
}

func TestParseArgsComponentShort(t *testing.T) {
	globalArgs, remainingArgs := command.ParseArgs([]string{"-c", "test-component", "test-command", "1", "2"})
	if globalArgs.Command != "test-command" {
		log.Fatalln("expecting test-command command after short component, got:", globalArgs.Command)
	}
	if globalArgs.Component != "test-component" {
		log.Fatalln("expecting test-component from short arg, got:", globalArgs.Component)
	}
	if !reflect.DeepEqual(remainingArgs, []string{"1", "2"}) {
		log.Fatalln("unexpected remaining args after short component arg, got:", remainingArgs)
	}
}

func TestParseArgsComponentLong(t *testing.T) {
	globalArgs, remainingArgs := command.ParseArgs([]string{"--component", "test-component", "test-command", "1", "2"})
	if globalArgs.Command != "test-command" {
		log.Fatalln("expecting test-command command after long component, got:", globalArgs.Command)
	}
	if globalArgs.Component != "test-component" {
		log.Fatalln("expecting test-component from short arg, got:", globalArgs.Component)
	}
	if !reflect.DeepEqual(remainingArgs, []string{"1", "2"}) {
		log.Fatalln("unexpected remaining args after short component arg, got:", remainingArgs)
	}
}

func TestGetComponentFromGit(t *testing.T) {
	component, err := command.GetComponentFromGit()
	if err != nil {
		log.Fatalln("error getting component name from git:", err)
	}
	if component != "cdflow2" {
		log.Fatalln("expected cdflow2 component from git, got:", component)
	}
}
