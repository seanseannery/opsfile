# opsfile (or ops)

## What is it?

It's a cli command tool, like `make` and `makefiles` but for standardizing and running live operations commands across the repo.

Simply create an `Opsfile` in your repo with common live operations commands you use and run it with `ops [env] <command>`.

ex
 * `ops preprod instance-count` - get the number of cloud instances in your preprod cluster
 * `ops prod open-dashboard` - open grafana/kibana/datadog/cloudwatch dashboard in your browser
 * `ops local logs` - tail logs for your local running docker environment to debug
 * `ops prod k8s -namespace` - runs kubectl top on the provided namespace

## Installation

**macOS / Linux** — paste this in your terminal:

```bash
curl -fsSL https://raw.githubusercontent.com/seanseannery/opsfile/main/install/install.sh | bash
```

The script detects your OS, downloads the correct binary from the [latest GitHub release](https://github.com/seanseannery/opsfile/releases/latest), and installs it to `/usr/local/bin/ops` (prompting for `sudo` if needed).

**Windows** — download `ops.exe` directly from the [releases page](https://github.com/seanseannery/opsfile/releases/latest).

## Usage

```
ops [flags] <environment> <command> [command-args]
```

| Flag | Short | Description |
|------|-------|-------------|
| `--dry-run` | `-d` | Print the resolved commands without executing them |
| `--silent` | `-s` | Execute commands without printing any output |
| `--version` | `-v` | Print the ops version and build platform, then exit |
| `--help` / `-?` | `-h` | Show this usage information, then exit |

### Opsfile format

Create an `Opsfile` in your repo root (or any parent directory):

```
# Variables — prefix with environment name to scope them
prod_AWS_ACCOUNT=1234567
preprod_AWS_ACCOUNT=8765431

# Commands — define per-environment shell lines
# Use "default" as a fallback when env-specific block is absent
tail-logs:
    default:
        aws cloudwatch logs --tail $(AWS_ACCOUNT)
    local:
        docker logs myapp --follow

list-instance-ips:
    prod:
        aws ec2 --list-instances
    preprod:
        aws ecs cluster --list-instances
```

Variable references use `$(VAR_NAME)`. Scoped variables (`prod_AWS_ACCOUNT`) take priority over unscoped ones (`AWS_ACCOUNT`).

## Why?

I really like Makefiles in my project repos.  It's a great way to build, test, spin-up local dev environments, and common CI actions without having to memorize the associated maven/docker/gradle/k8s specifics. Additionally, it makes it easy to onboard new engineers to the repo and share common CI scripts with teammates.

The other commands and scripts that often get reused and passed around are the operational tasks.  Checking logs, getting IP addresses or instance counts, etc. I get paged and I need to tail logs, or suppress an alarm, or get instance IPs. But Makefile isn't the right place for commands like that. Additionally, its not fun remember the 5 cli arguments kubectl needs to display logs for a specific container while my manager is asking for a status update on the outage.

So rather than copy/pasting gists with  bash aliases from team member to team member, or creating a "tools" repo with a bunch of adhoc scripts that doesn't get maintained, I thought I would create a tool, based on an established model, to make live operations on a repo or service more standardized and easier.

