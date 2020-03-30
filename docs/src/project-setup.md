---
name: Project Setup
route: /project-setup
---

# Project Setup

This guide covers setting up a project to use [cdflow2](/opensource/cdflow2). It assumes you have
already [installed cdflow2](installation).

## Git repository

To deploy a project with cdflow2 it needs a git repo. For example use the [GitHub CLI](https://cli.github.com/manual/gh_repo_create) to create a repo:

```shell
# change "myorg" and "myrepo" to something that makes sense for you
gh repo create myorg/myrepo
cd myrepo
```

## `cdflow.yaml` Boilerplate

Copy the following into a file in the root called `cdflow.yaml`:

```yaml
version: 2
team: TODO
config:
  image: TODO
  params:
builds:
terraform:
  image: hashicorp/terraform:0.12.23
```

Replace the value for `team` with the name of your team in [kebab case](https://wiki.c2.com/?KebabCase)
(i.e `lower-case-with-hyphens`).

See [cdflow.yaml reference](cdflow-yaml-reference).

## Config Image

The second `TODO` is the image used to configure your build & deployment. A good choice if you're starting
out deploying to AWS is:

```yaml
config:
  image: mergermarket/cdflow-config-aws-simple
```

Available config images include:

* [`mergermarket/cdflow2-config-aws-simple`](https://registry.hub.docker.com/r/mergermarket/cdflow2-config-aws-simple) -
  a config image for a simple setup with a single AWS account.
* [`mergermarket/cdflow2-config-aws-multi`](https://registry.hub.docker.com/r/mergermarket/cdflow2-config-aws-multi) -
  a config image for a setup wtih multiple teams deploying to multiple
  AWS accounts, for larger organisations.

## Configure Builds

If you need to one or more builds for your release - e.g. a docker container or lambda function - then
add to the `builds` section of cdflow.yaml. For example, to do a [docker](https://www.docker.com/) build and upload it to [ECR](https://aws.amazon.com/ecr/):

```yaml
builds:
  docker:
    image: mergermarket/cdflow-release-docker-ecr
```

Available build images include:

* [`mergermarket/cdflow2-release-docker-ecr`](https://registry.hub.docker.com/r/mergermarket/cdflow2-release-docker-ecr).
* [`mergermarket/cdflow2-release-lambda`](https://registry.hub.docker.com/r/mergermarket/cdflow2-release-lambda).

## Setup

Now that you've chosen a config image and build images, you can complete your setup interactively with:

```shell
cdflow2 setup
```

The options will depend on the chosen config container.

## Terraform

Terraform code should be placed in an `infra/` folder - by convention in the following files:

### `infra/variables.tf`

The following [variables](https://www.terraform.io/docs/configuration/variables.html) should be defined:

```terraform
variable "release" {
  type        = "map"
  description = "release metadata: version, commit, component & team"
}
```

In addition each build will result in additional map being provided - e.g. if you have have a
[`mergermarket/cdflow2-release-docker-ecr`](https://registry.hub.docker.com/r/mergermarket/cdflow2-release-docker-ecr) build under a "docker" key:

```terraform
variable "docker" {
  type        = "map"
  description = "release data from cdflow2-release-docker-ecr: image_id"
}
```

In addition you can declare additional environment specific variables - see below for how to provide the values.

### `infra/main.tf`

This is where you put your terraform code. In addition to the variables you've declared, you can also access the name of the the environment with the `terraform.workspace` value.

### `infra/outputs.tf`

Any [outputs](https://www.terraform.io/docs/configuration/outputs.html) you declare here will appear in the build output for your pipeline.

## Config

To provide the environment specific values of any config variables you have defined (i.e. in
`infra/variables.tf`), add a [JSON tfvars file](https://www.terraform.io/docs/configuration/variables.html#variable-definitions-tfvars-files) in `config/ENV.json` - for example:

### `config/live.json`

```JSON
{
  "my_config_name": "config value for the live environment"
}
```

## Build & Deploy

Once this one-off setup has been completed, you can build and deploy your software as follows:

```shell
# choose a version number:
version=1

# build the software
cdflow2 release $version

# deploy the software to an "aslive" environment:
cdflow2 deploy aslive $version

# deploy the software to a "live" environment:
cdflow2 deploy live $version
```

### Choosing a Version Number

It is recommended to use the build number from your CI server and the short git commit, separated by
a hyphen. This will allow you to identify the build and the commit that produced it. For example if
you're using Jenkins:

```shell
# for example "101-d37bfc2"
version="$BUILD_NUMBER-$(git rev-parse --short HEAD)"
```