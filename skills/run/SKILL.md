---
name: run
description: Run an ops command from the current repo's Opsfile. Accepts direct CLI arguments (/ops:run prod tail-logs) or natural language (/ops:run tail the production logs and grep for errors).
---

Run ops commands from the current repo's Opsfile. Handle both direct CLI invocations and natural language requests.

**Input:** `$ARGUMENTS`

## Behavior

### 1. Detect input type

If `$ARGUMENTS` begins with a CLI flag (e.g. `--`, `-`) or matches the pattern of ops positional arguments (e.g. `<environment> <command>`, or a known flag like `--list`), treat it as **direct CLI arguments**.

Otherwise, treat it as a **natural language request**.

### 2. Direct CLI mode

Run the command directly:

```bash
ops $ARGUMENTS
```

### 3. Natural language mode

a. First, discover what commands are available:

```bash
ops --list
```

b. Read the output to understand the available commands, environments, and descriptions.

c. Determine which command and environment best matches the user's intent. If the request is ambiguous or no command clearly matches, show the user the available options and ask them to clarify if they would like to choose a different ops command or attempt to generate a new (non-ops) cli command using a different tool that could fulfil their request.

d. Tell the user the command you resolved before running it:

```
I'll run: ops <environment> <command>
```

e. Execute it:

```bash
ops <resolved-environment> <resolved-command> [resolved-args]
```

f. If the skill ends up running a custom command because there wasnt one defined in Opsfile.  Ask if they would like to add what was just run as a new Opsfile command.

## Safety

- You **must not** issue commands (ops or otherwise) that would delete, destroy, remove anything in 'prod' or 'production' or any customer facing environments without direct user confirmation
- You **must** inform the user of any risk of outdated infrastructure-as-code when the user makes changes to infrastructure directly. Unless they dont utilize IaC for the project.
- You **must not** store, share, or print any infrastructure secrets in memory that may be used to gain access to critical systems. Critical Secrets should stored either in environment variables or .ops_secrets.env or the autheticated sessions of the underlying bash tools.

## Error handling

- If `ops` is not installed, direct the user to install it: https://github.com/seanseannery/opsfile
- If no Opsfile is found, explain that `ops` searches the current and parent directories for an `Opsfile`, and suggest `/ops:init-opsfile` to scaffold one for the current repo
- If the requested command or environment does not exist in the Opsfile, show the available options from `ops --list` and ask the user to clarify.
- If the ops command executes, but underlying bash commands fail to execute, suggest a potential solution for why the specific bash command failed. Investigate incorrectly set Opsfile variables.
