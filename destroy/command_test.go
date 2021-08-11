package destroy_test

import (
	"testing"

	"github.com/mergermarket/cdflow2/destroy"
)

func TestParseArgs(t *testing.T) {
	assertMatchArgs := func(t *testing.T, gotArgs, wantArgs *destroy.CommandArgs) {
		t.Helper()
		if gotArgs.EnvName != wantArgs.EnvName {
			t.Errorf("EnvName: got %s want %s", gotArgs.EnvName, wantArgs.EnvName)
		}
		if gotArgs.Version != wantArgs.Version {
			t.Errorf("Version: got %s want %s", gotArgs.Version, wantArgs.Version)
		}
		if gotArgs.PlanOnly != wantArgs.PlanOnly {
			t.Errorf("PlanOnly: got %t want %t", gotArgs.PlanOnly, wantArgs.PlanOnly)
		}
	}
	assertMatchBool := func(t *testing.T, got, want bool) {
		t.Helper()
		if got != want {
			t.Errorf("Bool: got %v want %v", got, want)
		}
	}

	t.Run("sad path - no args", func(t *testing.T) {
		args := []string{""}
		_, gotBool := destroy.ParseArgs(args)

		wantBool := false

		assertMatchBool(t, gotBool, wantBool)
	})

	t.Run("sad path - too many args", func(t *testing.T) {
		args := []string{"foo", "bar", "baz"}
		_, gotBool := destroy.ParseArgs(args)

		wantBool := false

		assertMatchBool(t, gotBool, wantBool)
	})

	t.Run("sad path set env", func(t *testing.T) {
		args := []string{"foo"}
		_, gotBool := destroy.ParseArgs(args)

		var result destroy.CommandArgs
		result.EnvName = "foo"
		wantBool := false

		assertMatchBool(t, gotBool, wantBool)
	})

	t.Run("set env + version", func(t *testing.T) {
		args := []string{"foo", "bar"}
		gotArgs, gotBool := destroy.ParseArgs(args)

		var result destroy.CommandArgs
		result.EnvName = "foo"
		result.Version = "bar"
		wantArgs, wantBool := &result, true

		assertMatchArgs(t, gotArgs, wantArgs)
		assertMatchBool(t, gotBool, wantBool)
	})

	t.Run("sad path set plan-only + env", func(t *testing.T) {
		args := []string{"-p", "foo"}
		_, gotBool := destroy.ParseArgs(args)

		var result destroy.CommandArgs
		result.EnvName = "foo"
		result.PlanOnly = true
		wantBool := false

		assertMatchBool(t, gotBool, wantBool)

		// alternate flag
		args = []string{"--plan-only", "foo"}
		assertMatchBool(t, gotBool, wantBool)
	})

	t.Run("set plan-only + env + version", func(t *testing.T) {
		args := []string{"-p", "foo", "bar"}
		gotArgs, gotBool := destroy.ParseArgs(args)

		var result destroy.CommandArgs
		result.EnvName = "foo"
		result.Version = "bar"
		result.PlanOnly = true
		wantArgs, wantBool := &result, true

		assertMatchArgs(t, gotArgs, wantArgs)
		assertMatchBool(t, gotBool, wantBool)
	})
}
