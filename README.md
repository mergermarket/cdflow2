![Unit tests](https://github.com/mergermarket/cdflow2/workflows/Unit%20tests/badge.svg)
[![Travis Build Status](https://www.travis-ci.org/mergermarket/cdflow2.svg?branch=master)](https://www.travis-ci.org/mergermarket/cdflow2)
![Build and publish release](https://github.com/mergermarket/cdflow2/workflows/Build%20and%20publish%20release/badge.svg)
[![Go Report Card](https://goreportcard.com/badge/github.com/mergermarket/cdflow2)](https://goreportcard.com/report/github.com/mergermarket/cdflow2)
![GitHub release (latest by date)](https://img.shields.io/github/v/release/mergermarket/cdflow2)
![GitHub Release Date](https://img.shields.io/github/release-date/mergermarket/cdflow2)
![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/mergermarket/cdflow2)
![GitHub](https://img.shields.io/github/license/mergermarket/cdflow2)

For usages documentation be refer to: https://developer-preview.acuris.com/opensource/cdflow2/

## Containers

`cdflow2` creates containers from the three images specified in your `cdflow.yaml`: one to perform configuration, one to create a release (i.e. for the `release` subcommand), and one to run `terraform`.

### Config

The config container is created from the image specified in `config_image` in `cdflow.yaml`. Its job is to apply any conventions (e.g. use of AWS or GCP, assuming roles in given accounts, etc) and setting up the environment for a relase or for terraform.

The container's entrypoint is invoked and communicated via a simple protocol over stdin and stdout. Any output to stderr from the container appears on stderr from cdflow2 so the user can see any warnings or errors. Requests to the config container are sent as json lines to its stdin stream and responses are returned as json lines from its stdout stream. The format of these messages are described here - the JSON examples here are formatted over multiple lines for readability:

#### `configure_release`

This allows the config container to set up the environment variables needed to create the release. For example if the release container needs to publish a docker image to an [ECR](https://aws.amazon.com/ecr/) repository in a particular AWS account, then the config container might use the AWS credentials from the environment and assume a role in that AWS account, passing through environment variables for the release container so that it publishes the image to the right place.

##### Example:

Request (contents of `env` are full set of environment variables `cdflow2` was invoked with):

```json
{
    "Action": "configure_release",
    "Config": {
        "key-from-config-in-cdflow-yaml": "value-from-config-in-cdflow-yaml"
    },
    "Env": {
        "ENV_NAME_FROM_CDFLOW2_INVOCATION": "ENV_VALUE_FROM_CDFLOW2_INVOCATION"
    }
}
```

Response:

```json
{
    "Env": {
        "ENV_NAME_FOR_RELEASE_CONTAINER": "ENV_VALUE_FOR_RELEASE_CONTAINER"
    }
}
```

#### `upload_release`

Once the release container has created and published some kind of release artefact (e.g. pushing a docker image to a docker registry), the config contianer is invoked again to store release meatadata (e.g. to an S3 bucket) - the config container is free to assume that `upload_release` will always follow `configure_release` and persist state in memory between.

`TerraformImage` contains the image digest for terraform so it can be stored to ensure the exact same image is used throughout the pipeline. `ReleaseMetadata` is a map of string keys to string values that will be passed to terraform as the `release` map variable.

The following keys will always be present in `ReleaseMetadata`:

* `commit` - the git commit cloned for the release.
* `version` - the version string passed on the command line to release.
* `component` - the name of the component, derived from the git repository name.

In addition any keys and values returned from the release container will be present (typically used to add details of the release artefact - e.g. docker image).

##### Example:

Request:

```json
{
    "Action": "upload_release",
    "TerraformImage": "hashicorp/terraform@sha256:aac4f7c61e8bd04c1ca14681b099cb8434788881bbe08febe5b7f9c0d2eabf1c",
    "ReleaseMetadata": {
        "commit": "3eaf7bcf155b7cd354083ab551ceb15d4290bebb",        
        "version": "101-3eaf7bcf",
        "component": "my-component",
        "image_id": "someco/my-component:101-3eaf7bcf"
    }
}
```

Response:

```json
{
    "Message": "Uploaded release metadata to s3://my-bucket/releases/101-3eaf7bcf.zip"
}
```

#### `prepare_terraform`

Invoked at the start of a command that uses terraform against an environment and release (e.g. deploy, destroy, shell). The config container is created with a volume mapped in `/release` (also the working directory), where it should download and unpack the release. The response data is then used to configure terraform.

##### Example:

Request:

```json
{
    "Action": "prepare_terraform",
    "Version": "101-3eaf7bcf",
    "Config": {
        "key-from-config-in-cdflow-yaml": "value-from-config-in-cdflow-yaml"
    },
    "Env": {
        "ENV_NAME_FROM_CDFLOW2_INVOCATION": "ENV_VALUE_FROM_CDFLOW2_INVOCATION"
    }
}
```

Response:

```json
{
    "TerraformImage": "hashicorp/terraform@sha256:aac4f7c61e8bd04c1ca14681b099cb8434788881bbe08febe5b7f9c0d2eabf1c",
    "Env": {
        "ENV_NAME_FOR_TERRAFORM_CONTAINER": "ENV_VALUE_FOR_TERRAFORM_CONTAINER"
    },
    "TerraformBackendType": "s3",
    "TerraformBackendConfig": {
        "bucket": "mybucket",
        "key": "path/to/my/key",
        "region": "eu-west-1"
    }
}
```

### Release

The config container is created from the image specified in `config_image` in `cdflow.yaml`. It is responsible for building and publishing a release package (e.g. a container image or lambda zip).

The container's entrypoint is invoked with the working directory (i.e. the source code) mapped in as the working directory and environment variables provided by the config container. Output from the container to stdout and stderr will be preserved, apart from the final line to stdout which should be a JSON object containing release metadata - this will be added to the `release` map variable passed to terraform (more info below).

### Terraform

A single terraform container is created from the image specified in the `terraform_image` key in `cdflow.yaml` and reused for each terraform command. In order to achieve this the entrypoint and command of the container is overridden to `/bin/sleep` and the actual terraform commands are exec'd in the container (i.e. the terraform image must include `/bin/sleep`, as the official images do).

#### Release

During release a terraform is run to download providers and modules - essentially running `terraform init infra/` and saving the downloaded providers and modules.

#### Backend configuration

When a non-release command is exected the config container is invoked to prepare terraform - downloading and unpacking the release (including the terraform providers and modules) and returning terraform backend config.

If it doesn't already exist then an empty backend [partial configuration](https://www.terraform.io/docs/backends/config.html#partial-configuration) is written to `infra/terraform.tf` with the backend type returned from the config container:

```hcl
terraform {
  backend "RETURNED_BACKEND_TYPE" {}
}
```

Terraform is then run to complete backend configuration based on the config keys and values returned from the config container - similar to the following:

```shell
terraform init \
    -get=false \
    -get-plugins=false \
    -backend-config="key1=value1" \
    -backend-config="key2=value2"
```

#### `deploy ENV VERSION`

Following backend configuration, deploy does a terraform plan and apply equivalent to the following:

```shell
export TF_IN_AUTOMATION=true

if terraform workspace list | grep -q '\bENV\b'; then
    terraform workspace select ENV
else
    terraform workspace new ENV
fi

terraform plan \
    -input=false \
    -var-file release-metadata-VERSION.json \
    -var-file config/ENV.json \
    -out=plan-TIMESTAMP \
    infra/

terraform apply \
    -input=false \
    plan-TIMESTAMP
```

Note that the only built in variables are the `release` map defined in the release metadata file (you can access the environment name via the workspace - i.e. `${terraform.workspace}`).


### Trivy
Trivy is a comprehensive and versatile security scanner. Trivy has scanners that look for security issues, and targets where it can find those issues.

Cdflow2 will automatically scan source code using Trivy and the mergermarket/cdflow2-trivy docker image.
The default configuration will search for Fixed Critical vulnerabilities, secrets and misconfigurations.
It also allows to configure some aspects on the fs and iamge scan when using mergermarket/cdflow2-build-docker-ecr.

```yaml

    trivy:
        image: mergermarket/cdflow2-trivy
        params:
            errorOnfindings: true #this is default can be set to false to 

```


## Documentation

https://developer-preview.acuris.com/opensource/cdflow2

## Subcommands

### release
Creates a cdflow2 releaase.
[Release subcommand docs.](/docs/src/commands/release.md)

### deploy
Deploys a cdflow2 release.
[Deploy subcommand docs.](/docs/src/commands/deploy.md)

### destroy
Destroys resources referenced in the /infra
[Destroy subcommand docs.](/docs/src/commands/destroy.md)

### shell
Runs cdflow2 interactive shell to give access to the underlying terraform command.
[Shell subcommand docs.](/docs/src/commands/shell.md)


