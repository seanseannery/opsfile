---
name: init-opsfile
description: Analyze the current repository's tech stack and generate a starter Opsfile with common local development and operations commands tailored to the detected platform.
---

Analyze the current repository and generate an `Opsfile` tailored to its tech stack and deployment platform. Customize specifically for this project's language, infrastructure, and environments. Always include a `local` environment at minimum. Never store passwords, tokens, or other secrets inline.

**Before starting:** read the Opsfile syntax specification and platform templates from the reference file installed alongside this skill: `${CLAUDE_PLUGIN_ROOT}/skills/init-opsfile/reference.md`

---

## Steps

### 1. Detect tech stack

Scan the repository root, documentation, and config for these signals:

**Documentation (read first for context):**
- `README.md`, `AGENTS.md`, `CLAUDE.md`, `Makefile`

**Language / Framework:**
- `go.mod` → Go
- `package.json` → Node.js (check `scripts` for framework hints)
- `pom.xml` or `build.gradle` → Java / JVM
- `requirements.txt`, `pyproject.toml`, or `setup.py` → Python
- `Cargo.toml` → Rust
- `Gemfile` → Ruby

**Containerization:**
- `Dockerfile` → Docker image build
- `docker-compose.yml` or `docker-compose.yaml` → Docker Compose local dev

**Deployment Platform (check in order of specificity):**
- `.github/workflows/` referencing `aws`, `ecs`, or `ecr` → AWS ECS
- `.github/workflows/` referencing `gke`, `cloud-run`, or `gcp` → GCP Cloud Run / GKE
- `.github/workflows/` referencing `azure`, `aks`, or `acr` → Azure
- `k8s/`, `kubernetes/`, `helm/` directories, or `kind: Deployment` in any YAML → Kubernetes
- No cloud signals detected → bare-metal / local only

**Infrastructure as Code (supplement or confirm platform detection):**
- `terraform/` or `*.tf` files → scan for `provider` blocks (`aws`, `google`, `azurerm`) to confirm cloud platform; scan `variable` and `locals` blocks, workspace names, and directory structure (e.g. `envs/prod/`, `environments/staging/`) to infer environment names
- `cloudformation/` or `*.yaml`/`*.json` containing `AWSTemplateFormatVersion` → AWS; scan `Parameters` and stack names for environment names
- `cdk/` or files importing `aws-cdk-lib` / `@aws-cdk` → AWS; scan stack class names and context variables for environment names
- `pulumi.yaml` or `Pulumi.*.yaml` → scan stack file names (e.g. `Pulumi.prod.yaml`, `Pulumi.staging.yaml`) for environment names and the `runtime` / provider config for platform
- `ansible/` or `*.playbook.yml` with `hosts:` blocks → bare-metal / SSH; scan inventory files for host group names as environment names

When IaC is found, use the detected environment names (e.g. `prod`, `staging`, `dev`) as the environment blocks in the generated Opsfile rather than generic placeholders.

If multiple platforms are detected, combine the most relevant sections. Always include a `local` environment.

### 2. Select and apply platform template

From `reference.md`, choose the template matching the detected platform. Substitute real values found in the repo wherever possible. Mark anything unknown with a `# TODO:` comment.

### 3. Generate the Opsfile

Produce an `Opsfile` customized for this specific repo:

- Substitute real detected values (cluster names, namespaces, log groups, service names) wherever possible
- Mark unknown values with `# TODO:` comments
- Include a `local` environment with at minimum: `tail-logs`, `check-resources`, `restart`
- Include `default:` blocks wherever the command works across environments
- Add a `#` comment above each command (shown as its description by `ops --list`)
- Use `$(VAR)` for any secrets; add a comment directing the user to `.ops_secrets.env`

### 4. Confirm and write

- Summarize the detected tech stack and which platform template was used
- Show the full generated Opsfile as a preview
- Ask the user to confirm before writing
- Write `Opsfile` to the repository root (or a location the user specifies)
- Run `ops --list` after writing to confirm it parses correctly
- Remind the user to add secret values to `.ops_secrets.env` and add that file to `.gitignore`
