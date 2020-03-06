package setup

import (
	"fmt"

	"github.com/mergermarket/cdflow2/command"
	"github.com/mergermarket/cdflow2/config"
	release "github.com/mergermarket/cdflow2/release/command"
)

// RunCommand runs the setup command.
func RunCommand(state *command.GlobalState, env map[string]string) (returnedError error) {

	// TODO check cdflow.yaml setup

	releaseRequirements, err := release.GetReleaseRequirements(state)
	if err != nil {
		return err
	}

	if err := config.Pull(state); err != nil {
		return err
	}

	configContainer, err := config.NewContainer(state, state.Manifest.Config.Image, "")
	if err != nil {
		return err
	}
	defer func() {
		if err := configContainer.Done(); err != nil {
			if returnedError != nil {
				returnedError = fmt.Errorf("%w, also %v", returnedError, err)
			} else {
				returnedError = err
			}
			return
		}
	}()

	return configContainer.Setup(state.Manifest.Config.Params, env, state.Component, state.Commit, state.Manifest.Team, releaseRequirements)
}
