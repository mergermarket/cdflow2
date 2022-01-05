# Design

`cdflow2` is a standalone binary program written in [Go](https://go.dev/). It is designed
to orchestrate building and storing artefacts and then deploying them into a set of
environments using [Terraform](https://www.terraform.io/) as part of a continuous delivery pipeline.

The core of `cdflow2` is quite small, using plugins running in [Docker](https://www.docker.com/)
containers to perform configuration, perform the builds and run terraform. This page describes
how these plugin containers work.

## Contents

## Config Plugin

You must choose one config container to use `cdflow2` via the `config` > `image` value in
[cdflow.yaml](cdflow-yaml-reference.md). It is responsible for:

* Performing interactive setup with the [setup command](commands/setup).
* Providing configuration for the build(s) at the start of the [release command](commands/release).
* Saving the release at the end of the [release command](commands/release).
* Retrieving the release at the start of a [deploy](commands/deploy), [destroy](commands/destroy) or [shell](commands/shell) command and providing the configuration to run Terraform.

A working example of a config container is https://github.com/mergermarket/cdflow2-config-acuris - this is only
suitable for building software within ION Analytics, but it may be useful to see how it works. It builds on
https://github.com/mergermarket/cdflow2-config-common, which is a common foundation for building
config containers.

The config container is started and the container entrypoint runs as a server accepting
remote procedure calls from `cdflow` - the mechanism for sending these RPCs is described
below. Input to and output from the container (i.e. what it reads from STDIN and writes
to STDOUT and STDERR) are sent directly to the user, so the config container is able to
run interactively.

### Remote Proceedure Calls (RPCs) Interface

In addition to the interface with the user via stdio, `cdflow2` makes calls to the config
container. It does this by using Docker to execute an additional command within the
container: `/app forward`. The job of this process is to forward requests and responses
to/from the main process within the container (the mechanism for this is outside the
scope of `cdflow2` itself, but the implementation in
[cdflow2-config-common](/opensource/cdflow2-config-common) does this via a server listening
on a UNIX domain socket within the container.

`cdflow2` sends requests as [JSON lines](https://jsonlines.org/) to STDIN of the forwarding
process, and receives responses as [JSON lines](https://jsonlines.org/) from STDOUT of the
forwarding process. The format of these lines is defined by the request and response
structures defined in
[`config/container.go`](https://github.com/mergermarket/cdflow2/blob/master/config/container.go)
and described below (each JSON document contains the listed fields).

Two eDocker volumes are also mapped into the config container for all commands except setup:

* `/release` - during the release command this is used to collect the information to save in the release. For the commands that run Terraform this is where the release is retrieved to.
* `/cache` - available as a place to cache data between runs.

### Setup RPC

The Setup RPC is invoked when the user runs the [`setup` command](commands/setup).

#### SetupRequest Properties

`Action`
: Always "setup".

`Commit`
: The id of the Git commit.

`Component`
: The name of the component inferred from the Git repo name (or passed explicitly by the user).

`Config`
: Config in [cdflow.yaml](cdflow-yaml-reference.md) under `config` > `params`.

`Env`
: The environment variables set for the main `cdflow2` process.

`ReleaseRequirements`
: This is a map of string arrays. The keys are the names of builds and the values are "needs" declared by each build (this is described in more detail in the ConfigureRelease RPC below).

#### SetupResponse Properties

`Success`
: Boolean value indicating success or failure.

### ConfigureRelease RPC

The ConfigureRelease RPC is invoked at the almost at start of the [release command](commands/release). There is first
a call to each build container to collect the "needs" of each build (an array of string identifiers), then this RPC
is invoked to provide the config for each build in the form of environment variables\n: for example a build that
needs to upload a lambda function would indicate a "lambda" need. The config container would then add environment
variables containing the AWS credentials and the name of the bucket to upload to. The exact details of these interfaces
are a contract between the build and config containers and are outside the scope of `cdflow2` itself.

#### ConfigureReleaseRequest Properties

`Action`
: Always "configure_release".

`Commit`
: The id of the Git commit.

`Component`
: The name of the component inferred from the Git repo name (or passed explicitly by the user).

`Config`
: Config in [cdflow.yaml](cdflow-yaml-reference.md) under `config` > `params`.

`Env`
: The environment variables set for the main `cdflow2` process.

`ReleaseRequirements`
: This is a map of string arrays. The keys are the names of builds and the values are "needs" declared by each build (described above).

`Version`
: The version string passed to the [release command](commands/release).

#### ConfigureReleaseResponse Properties

`AdditionalMetadata`
: String keys and values that are added to the `release` Terraform variable to make additional infomration available during deployment (e.g. the config container might receive a `team` parameter and want to make this available to the Terraform code).

`Env`
: A map of environment maps. The keys at the top level are the names of the builds (i.e. the keys user `builds` in [cdflow.yaml](cdflow-yaml-reference.md)) and the values are maps of environment variable names and values for each build.

`Success`
: Boolean value indicating success or failure.

### UploadRelease RPC

The UploadRelease RPC is invoked at the end of the [release command](commands/release) in order to persist the release
data collected in the `/release` volume. Since the container persists since the preceeding ConfigureRelease RPC the data
provided is not repeated and the configure container must hold onto what it needs.

#### UploadReleaseRequest Properties

`Action`
: Always "upload_release"

`TerraformImage`
: The unique Terraform docker image id to ensure that the same version of Terraform is always used with this release. The config container is responsible for adding this to the persisted release (not sure why this should be the config container rather than `cdflow2` adding it to the release volume directly - could be a future simplification of this interface).

#### UploadReleaseResponse Properties

`Message`
: A message to be displayed to the user (not sure why this is here since the config container can interact directly with the user now - could be removed as a future simplification of this interface).

`Success`
: Boolean value indicating success or failure.

### PrepareTerraform RPC

The PrepareTerraform RPC is invoked at the start of a command that runs Terraform (i.e.
[deploy](commands/deploy), [destroy](commands/destroy) or [shell](commands/shell)). Its job is to download the saved
release to the `/release` mapped volume, provide the [Terraform backend configuration](https://www.terraform.io/docs/language/settings/backends/configuration.html), and to provide environment variables passed to terraform (typically credentials used by Terraform providers).

#### PrepareTerraformRequest Properties

`Action`
: Always "prepare_terraform".

`Commit`
: The id of the Git commit.

`Component`
: The name of the component inferred from the Git repo name (or passed explicitly by the user).

`Config`
: Config in [cdflow.yaml](cdflow-yaml-reference.md) under `config` > `params`.

`Env`
: The environment variables set for the main `cdflow2` process.

`EnvName`
: The name of the environment passed to the command (e.g. [deploy](commands/deploy)).

`StateShouldExist`
: Optional boolean. Where present will cause validation that the statefile either does or doesn't exist (always provided for [deploy](commands/deploy)).

`Version`
: The version string passed to the command (e.g. [deploy](commands/deploy)).

#### PrepareTerraformResponse Properties

`Env`
: Map of environment variable names and values to use for the Terraform container.

`Success`
: Boolean value indicating success or failure.

`TerraformImage`
: The Terraform image identifier to run.

`TerraformBackendType`
:  The [Terraform backend type](https://www.terraform.io/docs/language/settings/backends/configuration.html#backend-types):  e.g. "s3".

`TerraformBackendConfig`
:  DEPRECATED: Old mechanism for passing Terraform backend config that didn't support hiding sensitive values.

`TerraformBackendConfigParameters`
:  Map of Terraform backend config parameters. Each value is a futher map containing `Value` and `DisplayValue`. `DisplayValue` should be provided where the value is sensitive (the display value will be displayed instead between square brackets to indicate it is a placeholder for the actual value).

## Build Plugins

[cdflow.yaml](cdflow-yaml-reference.md) can container zero or more named builds under the `builds` key. Each build
must have an `image` property, which is used to create a build container for that build. Build containers are
executed when the [release command](commnads/release) is run. They are often dependent on config
provided by a config container but are otherwise decoupled from them. The interface is much simpler than a config
container and as a result they are simpler to create.

The build container is invoked twice. The first time is to get a list of "needs" - string identfiers that are passed
to the config container to indicate what configuration is needed. This is a contract between the build and config
containers and which identifiers exist or what they mean is outside the scope of `cdflow2` itself. The second
invocation is to perform the build. This will typically be some kind of software build (e.g. building a Docker container)
and then uploading it to an archive (e.g. a Docker image registry). The build will then return information about
this artefact, which will be made available to Terraform in a variable with the same name as the build.

### Needs

The container's entrypoint is first invoked with a single "requirements" parameter. It must then write a JSON
document to STDOUT containing an array of string identifiers and then exit.

### Build

Once the config PrepareRelase RPC has been called the build container will be invoked again, this time without any
parameters. In addition to the environment variables provided by the config container, the following will also be
set:

`VERSION`
: The version string passed to the [release command](commands/release).

`COMPONENT`
: The name of the component inferred from the Git repo name (or passed explicitly by the user).

`COMMIT`
: The id of the Git commit.

`BUILD_ID`
: The name of the build in [cdflow.yaml](cdflow-yaml-reference.md).

`MANIFEST_PARAMS`
: The `params` key under the build in [cdflow.yaml](cdflow-yaml-reference.md) encoded in JSON.

The release volume will also be mapped within the container as `/build` so it can save data within the release. See
https://github.com/mergermarket/cdflow2-build-files for an example build plugin that makes use of this.

At the end of the build the container should write a `/release-metadata.json` file withing the container. The
keys and values within this JSON document will be provided as a Terraform map variable with the same name as
the build.

## Terraform Container

Terraform is run through a container. The image to use is configured in [cdflow.yaml](cdflow-yaml-reference.md) in
the `terraform` > `image` key. This might be an official
[Terraform Image from Hashicorp](https://hub.docker.com/r/hashicorp/terraform/), or might be a custom one
(e.g. based on an official image but with additional dependencies installed).

In addition to the Terraform binary the image must include `/bin/sleep` (the official ones do) - `cdflow2` reuses
the same Terraform container as an optimisation, this is used as the entrypoint with the Terraform commands exec'd
within the same container.