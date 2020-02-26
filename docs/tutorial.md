# Tutorial

## Summary

This guide will walk you through creating your first service based on [cdflow2](/) - a
simple static website hosted on AWS S3.

## Prerequisites

[cdflow2](../) will need to be installed - see the [installation instructions on the Getting Started page](../#installation).

The service we deploy will be running in AWS. You will need valid AWS credentials set
in environment variables:

* `AWS_ACCESS_KEY_ID`
* `AWS_SECRET_ACCESS_KEY`
* `AWS_SESSION_TOKEN` (only required if not using an IAM user).

We are also going to need the [hub command](https://hub.github.com/) for interacting with GitHub, and the
[AWS Command Line Interface](https://aws.amazon.com/cli/) for AWS. If you don't have those already you can install
them with:

```sh
brew install awscli hub
```

## Create a git repository and push to GitHub

Change directory to where you keep your projects, e.g.
   
```sh
cd ~/projects
```

Create a local git repository:
   
```
mkdir my-site
cd my-site
git init
```

Create a remote GitHub repository (replace with my-org/my-site to create within an organisation):

```sh
hub create --private my-site
```

Push some content:

```sh
echo Welcome to my site! > index.html
git add index.html
git commit -m 'first page'
git push
```

Congratulations, you now have a basic website pushed to GitHub.

## Add terraform infrastructure code

Create a folder called `infra/` (i.e. run `mkdir infra`) and add a [terraform config](https://www.terraform.io/docs/configuration/index.html) file called `main.tf` containing the following with the `website` local variable replaced with a unique website address (S3 bucket names and website domain names are globally unique):

```Terraform
locals {
    website = "my-site.com"
}

resource "aws_s3_bucket" "bucket" {
  bucket = local.website
  acl    = "public-read"
  policy = <<-POLICY
  {
    "Version":"2012-10-17",
    "Statement":[{
      "Effect":"Allow",
      "Principal": "*",
      "Action": "s3:GetObject",
      "Resource": "arn:aws:s3:::${local.website}/*"
    }]
  }
  POLICY

  website {
    index_document = "index.html"
    error_document = "error.html"
  }
}
```

## Add cdflow2 Configuration

Create the `cdflow.yaml` manifest file in the root of the project and add the following content:

```Yaml
version: 2
config:
  image: mergermarket/cdflow2-config-aws-simple
  params:
    # choose your preferred region here
    default_region: eu-west-1
terraform:
  image: hashicorp/terraform
```