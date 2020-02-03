package command_test

import (
	"log"
	"reflect"
	"testing"

	"github.com/mergermarket/cdflow2/command"
)

func TestParseArgs(t *testing.T) {
	var env command.GlobalEnvironment
	command, remainingArgs := command.ParseArgs([]string{"test-command", "1", "2", "3"}, &env)
	if command != "test-command" {
		log.Fatalln("expecting test-command command, got:", command)
	}
	if !reflect.DeepEqual(remainingArgs, []string{"1", "2", "3"}) {
		log.Fatalln("unexpected remaining args:", remainingArgs)
	}
}
func TestParseArgsEmpty(t *testing.T) {
	var env command.GlobalEnvironment
	command, remainingArgs := command.ParseArgs([]string{}, &env)
	if command != "" {
		log.Fatalln("expecting empty test-command command, got:", command)
	}
	if !reflect.DeepEqual(remainingArgs, []string{}) {
		log.Fatalln("unexpected empty remaining args, got:", remainingArgs)
	}
}

func TestParseArgsCommandOnly(t *testing.T) {
	var env command.GlobalEnvironment
	command, remainingArgs := command.ParseArgs([]string{"test-command"}, &env)
	if command != "test-command" {
		log.Fatalln("expecting empty test-command command, got:", command)
	}
	if !reflect.DeepEqual(remainingArgs, []string{}) {
		log.Fatalln("unexpected empty remaining args, got:", remainingArgs)
	}
}

func TestParseArgsComponentShort(t *testing.T) {
	var env command.GlobalEnvironment
	command, remainingArgs := command.ParseArgs([]string{"-c", "test-component", "test-command", "1", "2"}, &env)
	if command != "test-command" {
		log.Fatalln("expecting test-command command after short component, got:", command)
	}
	if env.Component != "test-component" {
		log.Fatalln("expecting test-component from short arg, got:", command)
	}
	if !reflect.DeepEqual(remainingArgs, []string{"1", "2"}) {
		log.Fatalln("unexpected remaining args after short component arg, got:", remainingArgs)
	}
}

func TestParseArgsComponentLong(t *testing.T) {
	var env command.GlobalEnvironment
	command, remainingArgs := command.ParseArgs([]string{"--component", "test-component", "test-command", "1", "2"}, &env)
	if command != "test-command" {
		log.Fatalln("expecting test-command command after long component, got:", command)
	}
	if env.Component != "test-component" {
		log.Fatalln("expecting test-component from short arg, got:", env.Component)
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
