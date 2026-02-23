# opsfile (or ops)

## What is it?

It's a cli command tool, like `make` and `makefiles` but for standardizing and running live operations commands across the repo.

Simply create an `Opsfile` in your repo with common live operations commands you use and run it with `ops [env] <command>`.

ex
 * `ops preprod instance-count` - get the number of cloud instances in your preprod cluster
 * `ops prod open-dashboard` - open grafana/kibana/datadog/cloudwatch dashboard in your browser
 * `ops local logs` - tail logs for your local running docker environment to debug
 * `ops prod k8s -namespace` - runs kubectl top on the provided namespace

## Why?

I really like Makefiles in my project repos.  It's a great way to build, test, spin-up local dev environments, and common CI actions without having to memorize the associated maven/docker/gradle/k8s specifics. Additionally, it makes it easy to onboard new engineers to the repo and share common CI scripts with teammates.

The other commands and scripts that often get reused and passed around are the operational tasks.  Checking logs, getting IP addresses or instance counts, etc. I get paged and I need to tail logs, or suppress an alarm, or get instance IPs. But Makefile isn't the right place for commands like that. Additionally, its not fun remember the 5 cli arguments kubectl needs to display logs for a specific container while my manager is asking for a status update on the outage.

So rather than copy/pasting gists with  bash aliases from team member to team member, or creating a "tools" repo with a bunch of adhoc scripts that doesn't get maintained, I thought I would create a tool, based on an established model, to make live operations on a repo or service more standardized and easier.

