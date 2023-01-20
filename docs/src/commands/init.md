---
name: Init
menu: Commands
route: /commands/init
---

# Init

## Usage

`cdflow2 [ GLOBALARGS ] init [ OPTS ]`

See [usage](./usage) for global options.

### Options:

`--name`
: Name of the new project repository

`--boilerplate`
: Git URL of the git repo to copy as boilerplate. To use a specific branch (or any valid git refspec), add "?ref=branch-name" to the end of the URL.

`--{boilerplate arguments}`
: Dynamic argument for the templates files. E.g.: `--domain name --account test`

## Description

### Init from basic template

Basic template contains only the bare minimum folder structure and necessary files.
To create a basic project run:
```shell
cdflow2 init --name project-name
```

### Init from boilerplate

Create a new project from a boilerplate repository and replace all the template variables with the provided values from command line arguments.

#### Supported boilerplates

##### Platform Team boilerplates
- [backend-router-boilerplate](https://github.com/mergermarket/backend-router-boilerplate)

  A backend router is a service which is meant to be an integration point for given backend services - only private services (i.e. not \*-subscriber) should be connected to it.
- [product-frontend-router-boilerplate](https://github.com/mergermarket/product-frontend-router-boilerplate)

  A product frontend service provides the traffic management for a product's user-facing services - i.e. ones used by subscribers and/or staff. This repo contains a boilerplate that can be used (e.g. by following our quick start guide) to create a product frontend service for your product.
- [node-minimal-boilerplate](https://github.com/mergermarket/node-minimal-boilerplate)

  This is a simple node boilerplate with no dependencies designed to get people up and running with cdflow as painlessly as possible.

##### Other Team boilerplates
- [goMakeIt](https://github.com/mergermarket/gomakeit)

  A walking skeleton of a go project using docker compose. Also has an example of an old style Jenkins Release pipeline.
- [node-library-boilerplate](https://github.com/mergermarket/node-library-boilerplate)

  A boilerplate for building a Node.js library
- [es6-express-boilerplate](https://github.com/mergermarket/es6-express-boilerplate)

  Boilerplate code for an Express.js app in ES2015 and Node 6 See the Node 6 docs to see what's available to play with.
- [react-express-boilerplate](https://github.com/mergermarket/react-express-boilerplate)

  A walking skeleton of a React project.

#### Boilerplates

To quickly get up and running you could use one of the supported boilerplates above, these should all be cdflow2 enabled and ready to deploy through aslive and live. 
You can also create your own boilerplate for future use.

#### Boilerplate variables

The boilerplates above use templating so that they remain generic and not team, product or project specific. The template variables look like `%{team}` and they are replaced when running init by adding a `--team SOME_VALUE` flag. You can use the `--help` flag for more information on these flags in the boilerplates.
