![The devzero logo](https://console.devzero.io/_next/image?url=%2F_next%2Fstatic%2Fmedia%2Ffull_logo.379052d8.png&w=1080&q=75)

![Workflow Status](https://img.shields.io/github/actions/workflow/status/devzero-inc/local-developer-analytics/go.yaml)
![License](https://img.shields.io/github/license/devzero-inc/local-developer-analytics)
![Version](https://img.shields.io/github/v/tag/devzero-inc/local-developer-analytics)

# ODA

The **ODA** CLI is local developer analytics tool. It provides:

- Tracking commands executed in user shell
- Tracking processes running on users computer
- Overview of command execution
- Overview of most intensive processes over time

## Installation

### Homebrew

You can install `oda` using [Homebrew][brew] (macOS or Linux):

```sh
brew install <>
```

### wget

You can install `oda` using `wget`:

```sh
wget https://github.com/devzero-inc/local-developer-analytics/releases/download/<version>/oda-<os>-<arch>.zip
unzip oda-<os>-<arch>.zip
mv oda /usr/local/bin
```

### Other methods

Install from source:

- Clone the project
- Change directory
- Run `make install`
  - binary will be in your `$GOPATH/bin`
  - if your PATH isn't set correctly: export GOPATH=$(go env GOPATH) && export PATH=$PATH:$GOPATH/bin
- Run `make install-global`
  - binary will be in `/usr/local/bin`

## Install & Usage

ODA help interface provides summaries for commands and flags:

```sh
oda --help
```

After binary has been compiled we can install the service with the following commands:

* `oda install` => This will install the daemon, configure base directory, and inject configuration into the shell
* `oda start` => This will start the daemon
* `oda stop` => This will stop the daemon
* `oda uninstall` => This will uninstall the ODA and remove all configuration
* `oda serve` => This will serve the local dashbaord with data overview

## Community

For updates on the ODA CLI, [follow this repo on GitHub][repo].

For information on contributing to this project, please see the [contributing guidelines](CONTRIBUTING.md).

[repo]: https://github.com/devzero-inc/local-developer-analytics