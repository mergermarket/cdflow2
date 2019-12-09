## cdflow.yaml

`cdflow.yaml` (or `cdflow.yml`) is a metadata file that is required to be in the top level of a cdflow project. The format of version 2 (i.e. if you run `cdflow2`) is described here.

### `version`

Must be exactly `2`.

### `config_image`

The docker image used to create the config container - described in more detail below. This container image will be pulled each time you invoke `cdflow2` within your component, so it is a good idea to pin it enough to be confident you won't have compatibility surprises as your pipeline runs through its stages (i.e. through multiple invocations of `cdflow2`).

### `release_image`

The docker image used to create the release container - described in more detail below. This container image will be pulled and executed only for the `release` subcommand - i.e. just once at the start of your pipeline (i.e. it is common to target a major version tag or `latest` to get automatic updates here).

### `terraform_image`

The docker image used to create the terraform container - described in more detail below. The exact tag for this image is stored with the release, so you can be confident it will not change as that release version is promoted through your pipeline (i.e. you won't pick up a new version of Terraform for the first time when you are deploying your release into production).

## Containers

`cdflow2` creates and uses three containers: one to perform configuration, one to create a release (i.e. for the `release` subcommand), and one to run `terraform`.

### Config

The config container is created from the image specified in `config_image` in `cdflow.yaml`. Its job is to apply any conventions (e.g. use of AWS or GCP, assuming roles in given accounts, etc) and setting up the environment for a relase or for terraform.

The container's entrypoint is invoked and communicated via a simple protocol over stdio. Any output to stderr from the container appears on stderr from cdflow2 so the user can see any warnings or errors. Requests to the config container are sent as json lines to its stdin stream and responses are returned as json lines from its stdout stream. The format of these messages are described here - the JSON documents are formatted over multiple lines for readability:

#### `configure_release`

This allows the config container to set up the environment variables needed to create the release. For example if the release container needs to publish a docker image to an [ECR](https://aws.amazon.com/ecr/) repository in a particular AWS account, then the config container might use the AWS credentials from the environment and assume a role in that AWS account, passing through environment variables for the release container so that it publishes the image to the right place.

##### Example:

Request (contents of `env` are full set of environment variables `cdflow2` was invoked with):

```json
{
    "action": "configure_release",
    "config": {
        "key-from-config-in-cdflow-yaml": "value-from-config-in-cdflow-yaml"
    },
    "env": {
        "ENV_NAME_FROM_CDFLOW2_INVOCATION": "ENV_VALUE_FROM_CDFLOW2_INVOCATION"
    }
}
```

Response:

```json
{
    "env": {
        "ENV_NAME_FOR_RELEASE_CONTAINER": "ENV_VALUE_FOR_RELEASE_CONTAINER"
    }
}
```

#### `publish_release`

Once the release container has created and published some kind of release artefact (e.g. pushing a docker image to a docker registry), the config contianer is invoked again to store release meatadata (e.g. to an S3 bucket) - the config container is free to assume that `publish_release` will always follow `configure_release` and persist state in memory between.

### Release



### Terraform



## Subcommands

### release


### deploy


### destroy


### shell

