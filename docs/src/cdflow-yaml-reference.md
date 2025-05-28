---
name: cdflow.yaml Reference
route: /cdflow-yaml-reference
---

# `cdflow.yaml`

cdflow.yaml is a metadata file you place in the root of your project
controlling how cdflow2 builds and deploys your code.

## Full Example

```yml
# required - always "2" for cdflow2
version: 2

# optional - default is set to "config/"
config_files_folder: "config/"

# required - descibed below
config:
  image: mergermarket/cdflow2-config-aws-simple
  params:
    # parameters are specific to the config image you are using
    default-region: eu-west-1

# how to build your code - described below
builds:
  docker:
    image: mergermarket/cdflow2-build-docker-ecr
    # optional - list of env var that key and value should
    # be copied to the build/release container
    env_vars:
      - FOO
      - BAR
  lambda:
    image: mergermarket/cdflow2-build-lambda
    params:
      # some build images (like this one) take additional
      # configuration parameters (it's docker all the way down
      # for this image!)
      image: node:12
      cmd: npm run build

# required - the terraform docker image to use
terraform:
  image: hashicorp/terraform:0.12.23
```

## Reference

### `version` (required)

For cdflow2 this should always be the string value `2` - this is to
prevent you accidentally using the wrong version. For example:

```yaml
version: 2
```

### `config_files_folder` (optional)
Folder where the enviroment specific configuration files and common.json are stored. Default is `config/`

```yaml
config_files_folder: "infra_config/"
```

### `config` (required)

Config is used to select and configure the container that sets up the
environment for building and deploying the service. For example:

```yaml
config:
  image: mergermarket/cdflow2-config-aws-simple
  params:
    # parameters are specific to the config image you are using
    default-region: eu-west-1
```

#### `config > image` (required)

The docker image to use for config. Examples include:

* [`mergermarket/cdflow2-config-aws-simple:latest`](https://registry.hub.docker.com/r/mergermarket/cdflow2-config-aws-simple) -
  a config image for a simple setup with a single AWS account.
* [`mergermarket/cdflow2-config-acuris:latest`](https://registry.hub.docker.com/r/mergermarket/cdflow2-config-acuris) -
  a config image teams deploying with Acuris infrastructure.

#### `config > params` (optional)

A dictionary of parameters passed to the config container. What
parameters are supported depends on the config image used, so check
the specific documentation for that image.

### `builds` (optional)

Builds contains a dictionary of named builds that will be built when you
run `cdflow2 release` - e.g. building a docker image or lambda zip. For
example:

```yaml
builds:
  docker:
    image: mergermarket/cdflow2-build-docker-ecr
```

#### `builds > [name] > image` (required)

The image used to do the build. Examples include:

* [`mergermarket/cdflow2-release-docker-ecr:latest`](https://registry.hub.docker.com/r/mergermarket/cdflow2-release-docker-ecr)
* [`mergermarket/cdflow2-release-lambda:latest`](https://registry.hub.docker.com/r/mergermarket/cdflow2-release-lambda)

#### `builds > [name] > params` (optional)

A dictionary of parameters passed to the build container. What parameters
are supported depends on the build container used, so check the specific
documentation for that image.

#### `builds > [name] > env_vars` (optional)

The builds.[name].env_vars configuration allows specific environment variables to be injected into the release/build container at runtime.

This was needed because some builds require access to artifacts behind secure authentication.

Note: this config does not define values for the variables. Instead, it copies the specified environment variables from the build environment into the container. For example, if the build environment has FOO_APIKEY=secret123 and env_vars includes FOO_APIKEY, then the build container will also have FOO_APIKEY=secret123 set.

In most cases where this is used, the environment variables will also need to be passed to the container securely. To do this, you'll need to add the required secrets to your params. Here's a generic example:

```
builds:
  docker:
    image: mergermarket/cdflow2-build-docker-ecr
    params:
      buildx: true
      platforms: linux/arm64,linux/amd64
      secrets:
        - 'id=FOO_APIKEY,env=FOO_APIKEY'
    env_vars:
      - FOO_APIKEY
```

More information can be found here:
https://docs.docker.com/build/building/secrets/#sources

### `terraform > image` (required)

The [terraform docker image](https://registry.hub.docker.com/r/hashicorp/terraform)
to use to run [Terraform](https://www.terraform.io/). The exact docker
image will be maintained through the pipeline, but it is probably still
a good idea to pin the version since once you update a state file with
a newer version (e.g. with a `terraform apply` when you `cdflow2 deploy`)
it is not possible to go back to the older version. For example:

```yaml
terraform:
  image: hashicorp/terraform:0.12.24
```

See [latest hashicorp/terraform tags on Docker Hub](https://registry.hub.docker.com/r/hashicorp/terraform/tags).