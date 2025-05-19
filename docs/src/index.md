---
name: Overview
route: /
navigation:
  - name: Installation
    url: installation
    icon: Download
  - name: Project Setup
    url: project-setup
    icon: Settings
  - name: cdflow.yaml Reference
    url: cdflow-yaml-reference
    icon: Description
  - name: Usage
    url: commands/usage
    icon: Info
  - name: Init Command
    url: commands/init
    icon: Init
  - name: Setup Command
    url: commands/setup
    icon: Edit
  - name: Release Command
    url: commands/release
    icon: Add
  - name: Deploy Command
    url: commands/deploy
    icon: Forward
  - name: Destroy Command
    url: commands/destroy
    icon: Delete
  - name: Shell Command
    url: commands/shell
    icon: AttachMoney
  - name: Common Terraform Setup
    url: common-terraform-setup
    icon: Computer
  - name: Design
    url: design
    icon: Architecture
---

# Overview

[cdflow2](/opensource/cdflow2) is an open source tool for managing services and infrastructure using [terraform](https://terraform.io), following the principles of *continuous delivery*.

A typical pipeline consists of a [release step](commands/release) where your software is built
and the build is published somewhere (the software is "released"), followed by one or more
[deploy steps](commands/deploy) where Terraform updates an environment to use that release.

## Next Steps

* [Install cdflow2](installation)
* [Project Setup](project-setup)
* [cdflow.yaml Reference](cdflow-yaml-reference)
* [cdflow2 Design](design)
