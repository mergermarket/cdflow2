package command_test

import (
	"log"
	"reflect"
	"testing"

	"github.com/mergermarket/cdflow2/command"
)

func TestParseArgs(t *testing.T) {
	globalArgs, remainingArgs, err := command.ParseArgs([]string{"test-command", "1", "2", "3"})
	if err != nil {
		log.Fatalln("unexpected error from parseArgs:", err)
	}
	if globalArgs.Command != "test-command" {
		log.Fatalln("expecting test-command command, got:", globalArgs.Command)
	}
	if !reflect.DeepEqual(remainingArgs, []string{"1", "2", "3"}) {
		log.Fatalln("unexpected remaining args:", remainingArgs)
	}
}
func TestParseArgsEmpty(t *testing.T) {
	globalArgs, remainingArgs, err := command.ParseArgs([]string{})
	if err != nil {
		log.Fatalln("unexpected error from parseArgs:", err)
	}
	if globalArgs.Command != "" {
		log.Fatalln("expecting empty test-command command, got:", globalArgs.Command)
	}
	if !reflect.DeepEqual(remainingArgs, []string{}) {
		log.Fatalln("expected empty remaining args, got:", remainingArgs)
	}
}

func TestParseArgsCommandOnly(t *testing.T) {
	globalArgs, remainingArgs, err := command.ParseArgs([]string{"test-command"})
	if err != nil {
		log.Fatalln("unexpected error from parseArgs:", err)
	}
	if globalArgs.Command != "test-command" {
		log.Fatalln("expecting empty test-command command, got:", globalArgs.Command)
	}
	if !reflect.DeepEqual(remainingArgs, []string{}) {
		log.Fatalln("expected empty remaining args, got:", remainingArgs)
	}
}

func TestParseArgsComponentShort(t *testing.T) {
	globalArgs, remainingArgs, err := command.ParseArgs([]string{"-c", "test-component", "test-command", "1", "2"})
	if err != nil {
		log.Fatalln("unexpected error from parseArgs:", err)
	}
	if globalArgs.Command != "test-command" {
		log.Fatalln("expecting test-command command after short component, got:", globalArgs.Command)
	}
	if globalArgs.Component != "test-component" {
		log.Fatalln("expecting test-component from short arg, got:", globalArgs.Command)
	}
	if !reflect.DeepEqual(remainingArgs, []string{"1", "2"}) {
		log.Fatalln("unexpected remaining args after short component arg, got:", remainingArgs)
	}
}

func TestParseArgsComponentLong(t *testing.T) {
	globalArgs, remainingArgs, err := command.ParseArgs([]string{"--component", "test-component", "test-command", "1", "2"})
	if err != nil {
		log.Fatalln("unexpected error from parseArgs:", err)
	}
	if globalArgs.Command != "test-command" {
		log.Fatalln("expecting test-command command after long component, got:", globalArgs.Command)
	}
	if globalArgs.Component != "test-component" {
		log.Fatalln("expecting test-component from short arg, got:", globalArgs.Component)
	}
	if !reflect.DeepEqual(remainingArgs, []string{"1", "2"}) {
		log.Fatalln("unexpected remaining args after component arg, got:", remainingArgs)
	}
}

func TestParseArgsComponentLongWithEquals(t *testing.T) {
	globalArgs, remainingArgs, err := command.ParseArgs([]string{"--component=test-component", "test-command", "1", "2"})
	if err != nil {
		log.Fatalln("unexpected error from parseArgs:", err)
	}
	if globalArgs.Command != "test-command" {
		log.Fatalln("expecting test-command command after long component, got:", globalArgs.Command)
	}
	if globalArgs.Component != "test-component" {
		log.Fatalln("expecting test-component from short arg, got:", globalArgs.Component)
	}
	if !reflect.DeepEqual(remainingArgs, []string{"1", "2"}) {
		log.Fatalln("unexpected remaining args after component arg, got:", remainingArgs)
	}
}

func TestParseArgsCommitLong(t *testing.T) {
	globalArgs, remainingArgs, err := command.ParseArgs([]string{"--commit", "test-commit", "test-command", "1", "2"})
	if err != nil {
		log.Fatalln("unexpected error from parseArgs:", err)
	}
	if globalArgs.Command != "test-command" {
		log.Fatalln("expecting test-command command after long component, got:", globalArgs.Command)
	}
	if globalArgs.Commit != "test-commit" {
		log.Fatalln("expecting test-commit from short arg, got:", globalArgs.Commit)
	}
	if !reflect.DeepEqual(remainingArgs, []string{"1", "2"}) {
		log.Fatalln("unexpected remaining args after commit arg, got:", remainingArgs)
	}
}

func TestParseArgsCommitLongWithPrefix(t *testing.T) {
	globalArgs, remainingArgs, err := command.ParseArgs([]string{"--commit=test-commit", "test-command", "1", "2"})
	if err != nil {
		log.Fatalln("unexpected error from parseArgs:", err)
	}
	if globalArgs.Command != "test-command" {
		log.Fatalln("expecting test-command command after long component, got:", globalArgs.Command)
	}
	if globalArgs.Commit != "test-commit" {
		log.Fatalln("expecting test-commit from short arg, got:", globalArgs.Commit)
	}
	if !reflect.DeepEqual(remainingArgs, []string{"1", "2"}) {
		log.Fatalln("unexpected remaining args after commit arg, got:", remainingArgs)
	}
}

func TestParseArgsVersion(t *testing.T) {
	globalArgs, _, err := command.ParseArgs([]string{"--version"})
	if err != nil {
		log.Fatalln("unexpected error from parseArgs:", err)
	}
	if globalArgs.Command != "version" {
		log.Fatalln("expected command to be version, got:", globalArgs.Command)
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
