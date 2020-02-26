# Setup AWS

[cdflow2](/) is a thin wrapper around [Terraform](https://terraform.io/). The point of using
[cdflow2](/) rather than Terraform directly is to do some common setup and follow agreed
best practices across your organisation.

[cdflow2](/) itself is cloud provider agnostic, but this guide is going to cover a simple
way of preparing to deploy to [AWS](https://aws.amazon.com/), where all of the resources
are going to live in a single AWS account. This is great for getting up and running quickly,
but there are also other options that may be prefered (e.g. splitting your production and
non-production infrastructure into separate accounts). This is based on using the
[mergermarket/cdflow2-config-aws-simple](https://hub.docker.com/repository/docker/mergermarket/cdflow2-config-aws-simple)
config container - for more detailed instructions see the documentation there. If this doesn't
meet your needs then you may be able to find (or create) a config container for your cloud provider/setup.

*NOTE: if your organisation already uses cdflow2, this initial setup will not be required. Speak
to your local platform team for the configuration details for your setup.*

## Prerequisites

You will need the [AWS Command Line Interface](https://aws.amazon.com/cli/) configured with
valid credentials.

To install:

```sh
brew install awscli
```

To configure credentials set the `AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY` and possibly
the `AWS_SESSION_TOKEN` environment variables, or run the following to configure the CLI:

```sh
aws configure
```

You should also decide which AWS region you are going to deploy to - see this
[list of valid region codes](https://docs.aws.amazon.com/general/latest/gr/rande.html#regional-endpoints)
- e.g. `eu-west-1`, `us-east-1`, etc.

## Create tfstate Bucket

Terraform can use a backend to store its state (see
[State Storage and Locking](https://www.terraform.io/docs/backends/state.html)
in the Terraform docs) - essentially a mapping of the logical resources
you define in your infrastructure definition and the physical resources that 
Terraform creates.

[mergermarket/cdflow2-config-aws-simple](https://hub.docker.com/repository/docker/mergermarket/cdflow2-config-aws-simple)
applies the convention that this is stored in an [S3 bucket](https://www.terraform.io/docs/backends/types/s3.html)
identified by following the naming convention:

    cdflow2-tfstate-...

Check if you already have a bucket that follows the convention with:

```sh
aws s3 ls s3:// | grep cdflow2-tfstate-
```

If not create one with the following (replace `REGION` with your chosen region's identifier):

```sh
bucket="cdflow-tfstate-$(date +%s)$RANDOM"

aws s3 mb --region=REGION "s3://$bucket"

aws s3api put-bucket-versioning \
    --bucket "$bucket" \
    --versioing-configuration Status=Enabled
```

Versioning is important for a tfstate bucket, so make sure each command succeeds.

## Create Terraform Locks Table

In order to 

## Create Release Bucket

[cdflow2](/) stores information about each release in order to keep deploys of that
release consistent.
[mergermarket/cdflow2-config-aws-simple](https://hub.docker.com/repository/docker/mergermarket/cdflow2-config-aws-simple)
stores these releases in S3 in a "release bucket". As with the tfstate bucket, these are based on a naming convention:

    cdflow2-releases-...

Check for existing bucket:

```sh
aws s3 ls s3:// | grep cdflow2-releases-
```

If you don't have one, make a bucket (replace `REGION` with your chosed region):

```sh
aws s3 mb --region=REGION "s3://cdflow-releases-$(date +%s)$RANDOM"
```

Releases are written once and never updated, so no need to make it a versioned bucket.