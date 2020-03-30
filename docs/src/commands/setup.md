---
name: Setup
menu: Commands
route: /setup
---

# Setup

## Usage

```
cdflow2 [ GLOBALARGS ] setup

Global args:

  --component COMPONENT_NAME   - override component name (inferred from git by default).
  --commit GIT_COMMIT          - override the git commit (inferred from git by default).
  --no-pull-config             - don't pull the config container (must exist).
  --no-pull-release            - don't pull the release container (must exist).
  --no-pull-terraform          - don't pull the terraform container (must exist).
  --quiet | -q                 - hide verbose description of what's going on.
  --version                    - print the version number and exit.
  --help                       - print the help message and exit.
```

## Description

Setup can be used to check the project setup and perform any additional interactive setup.