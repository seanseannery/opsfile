# opsfile (aka `ops`)

## What does `ops` do?

  It's a cli tool, essentially like `make` and `makefiles` but for sharing and executing live-operations / on-call commands for the repo.  Simply create an `Opsfile` in your repo with common on-call commands your team uses and run it with `ops [env] <command>`.

## Installation

  ### Homebrew (MacOS / Linux)

  ```bash
  brew tap seanseannery/opsfile https://github.com/seanseannery/opsfile
  brew install seanseannery/opsfile/opsfile
  ```
  After tapping, `brew upgrade seanseannery/opsfile/opsfile` keeps `ops` up to date.

  ### npm (MacOS / Linux)

  ```bash
  npm install -g github:seanseannery/opsfile
  ```
  Requires Node.js ≥ 14. Downloads the correct platform binary from the latest GitHub release on install.
  To upgrade, re-run the same command. To uninstall: `npm uninstall -g opsfile`.

  ### curl (MacOS / Linux)

  ```bash
  curl -fsSL https://raw.githubusercontent.com/seanseannery/opsfile/main/install/install.sh | bash
  ```
  Detects your OS, downloads the correct binary from the [latest GitHub release](https://github.com/seanseannery/opsfile/releases/latest), and installs to `/usr/local/bin/ops`.

  ### Windows
  Download `ops.exe` directly from the [releases page](https://github.com/seanseannery/opsfile/releases/latest).


## Why use it?

  1. **Less Stress:** quickly find logs during a live outage instead of having to google that k8s flag you keep forgetting.
  2. **Reduced agentic context / token usage:** have agents run ops scripts instead of googling the correct aws cli command every time.
  3. **Knowledge Sharing:** Share common commands with your team for easier onboarding and debugging
  4. **Encapsulation:** keep your Makefile small and focused on CI/CD tasks

  Makefiles are a great way to build, test, spin-up local dev environments, and other common CI actions without having to memorize the associated maven/docker/npm/gradle/k8s specifics. Additionally, makefiles make it easy to onboard new engineers to the repo and share common CI scripts with teammates.

  In my experience, the other commands and scripts that often get passed around are the operational/on-call tasks.  Checking logs, getting IP addresses or instance counts, viewing dashboards, etc. I get paged and I need to tail logs, or suppress an alarm, or get instance IPs under pressure. But it's not fun remembering the 5 cli arguments kubectl needs to display logs for a specific container while my manager is asking for a status update on the outage. Additionally, Makefiles aren't the right place for non-ci commands like that.

  So rather than copy/pasting gists with bash aliases from team member to team member, or creating a "tools" repo with a bunch of adhoc scripts that doesn't get maintained, I thought I would create a tool, based on an established model (`make`), to improve live operations on a service and make it more standardized, shareable, and easier.


## Getting started

  ### Step 1: Opsfile 

  Create an `Opsfile` in your repo root.  Its just like makefile syntax. Below is a simple example.

  ```make
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

  # Shell environment variables are injected automatically using the same $(VAR) syntax.
  # No declaration needed — if ops cant find in the Opsfile, it will fall back to env variables
  show-profile:
      default:
          echo "Using AWS profile: $(AWS_PROFILE)"
  ```

  > [!TIP] 
  > There are more sample Opsfile examples in the `/examples` folder.

  > [!WARNING] 
  > Be sure not to include any secrets or access keys into your Opsfile as they could get shared visibly once committed to the repo.  Instead you can inject your secrets into the Opsfile via environment variables if needed.

  ### Step 2: Call the `ops` CLI

    ```bash
    ops [flags] <your_environment> <your_command> [any-command-args]
    ```

    | Flag | Short | Description |
    |------|-------|-------------|
    | `--directory <path>` | `-D <path>` | Use the Opsfile in the given directory instead of searching parent directories |
    | `--dry-run` | `-d` | Print the resolved commands without executing them |
    | `--silent` | `-s` | Execute commands without printing output |
    | `--version` | `-v` | Print the ops version, commit, and build platform, then exit |
    | `--help` | `-?`, `-h` | Show usage information, then exit |
  
  #### Examples:
  * `ops preprod instance-count` - get the number of cloud instances in your preprod cluster
  * `ops prod open-dashboard` - open grafana/kibana/datadog/cloudwatch dashboard in your browser
  * `ops local logs` - tail logs for your local running docker environment to debug
  * `ops prod k8s -namespace myspace` - runs kubectl top on the provided namespace

  > [!TIP] 
  > If you want, you could even alias the ops+env command in your terminal profile to make it even more user friendly
  >```bash
  >$> alias prod="ops prod"
  >$> prod tail-logs
  >```

  > [!TIP] 
  > To save even more AI tokens / context, you can create a skill or plugin for your repo, or include command details in AGENTS.md or CLAUDE.md
  >```bash
  >$> claude 'analyze my projects Opsfile and add a table of commands and supported environments to CLAUDE.md for operational debugging'
  >```

## CONTRIBUTING

For tips on how to set up dev environment, build, test, and how to follow PR and community best practices. Please read [CONTRIBUTING.md](CONTRIBUTING.md)