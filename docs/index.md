# Getting Started

[cdflow2](/) is an open source tool for managing services and infrastructure using
using [terraform](https://terraform.io), following the principles of *continuous delivery*.

## Installation

### Mac with Homebrew

If you're a [Homebrew](https://brew.sh/) user, installation is as simple as:

```sh
brew install mergermarket/tap/cdflow2
```

To upgrade:

```sh
brew upgrade mergermarket/tap/cdflow2
```

Also available, but not recommended!

```sh
brew remove mergermarket/tap/cdflow2
```

### Other

Download the [latest release from GitHub](https://github.com/mergermarket/cdflow2/releases).

## Running cdflow2

To check your installation, run:

```sh
cdflow2
```

You should see a usage message.

## Configure your cloud provider

Before you can start provisioning infrastructure, you need to create some supporting resources in your cloud provider:

* [Setup AWS](aws/)