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

	assertMatchError := func(t *testing.T, err error, wantError bool) {
		t.Helper()
		if wantError {
			if err == nil {
				t.Error("Error expected, but got nil")
			}
		} else if err != nil {
			t.Errorf("Error not expected, but got %v", err)
		}
	}

	t.Run("sad path - no args", func(t *testing.T) {
		args := []string{""}
		_, err := destroy.ParseArgs(args)

		assertMatchError(t, err, true)
	})

	t.Run("sad path - too many args", func(t *testing.T) {
		args := []string{"foo", "bar", "baz"}
		_, err := destroy.ParseArgs(args)

		assertMatchError(t, err, true)
	})

	t.Run("sad path set env", func(t *testing.T) {
		args := []string{"foo"}
		_, err := destroy.ParseArgs(args)

		assertMatchError(t, err, true)
	})

	t.Run("set env + version", func(t *testing.T) {
		args := []string{"foo", "bar"}
		gotArgs, err := destroy.ParseArgs(args)

		wantArgs := &destroy.CommandArgs{
			EnvName: "foo",
			Version: "bar",
		}

		assertMatchArgs(t, gotArgs, wantArgs)
		assertMatchError(t, err, false)
	})

	t.Run("sad path set plan-only + env", func(t *testing.T) {
		args := []string{"-p", "foo"}
		_, err := destroy.ParseArgs(args)

		assertMatchError(t, err, true)
	})

	t.Run("set plan-only + env + version", func(t *testing.T) {
		args := []string{"-p", "foo", "bar"}
		gotArgs, err := destroy.ParseArgs(args)

		wantArgs := &destroy.CommandArgs{
			EnvName:  "foo",
			Version:  "bar",
			PlanOnly: true,
		}

		assertMatchArgs(t, gotArgs, wantArgs)
		assertMatchError(t, err, false)
	})
}
