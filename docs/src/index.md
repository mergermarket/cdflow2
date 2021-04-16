---
name: Overview
route: /
---

# Overview

[cdflow2](/) is an open source tool for managing services and infrastructure using
using [terraform](https://terraform.io), following the principles of *continuous delivery*.

A typical pipeline consists of a [release step](commands/release) where you build your binary (once), followed
by one or more [deploy steps](commands/deploy) where you use Terraform to update an environment to use that
release.

## Next Steps

* [Install cdflow2](installation)
* [Project Setup](project-setup)
