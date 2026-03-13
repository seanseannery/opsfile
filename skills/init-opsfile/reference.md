# Opsfile syntax reference

An Opsfile is a plain-text file (no extension) placed at the repository root. It follows similar, but not exact syntax as Makefiles.

## Variables

```
NAME=value                      # unscoped — available in all environments
prod_NAME=value                 # env-scoped — only resolved when env is "prod"
```

Variable references use `$(NAME)` syntax. The resolver checks (in order):
1. Opsfile env-scoped (`prod_NAME`)
2. Shell env-scoped (`prod_NAME` from environment)
3. Opsfile unscoped (`NAME`)
4. Shell unscoped (`NAME` from environment)

Non-identifier tokens like `$(shell date +%s)` pass through unchanged as subshell expressions.

Secrets must never be defined inline. Use `$(SECRET_VAR)` and inject at runtime:
- via `.ops_secrets.env` (auto-loaded if present next to the Opsfile)
- via `ops -e .env.prod <env> <command>`

## Commands

```
# Comment above a command becomes its description in `ops --list`
command-name:
    environment-name:
        shell line one
        shell line two \
            continued
    default:
        fallback shell line
```

Rules:
- Command names: non-indented, end with `:`
- Environment blocks: one level of indentation, end with `:`
- Shell lines: two levels of indentation
- `default:` is used as a fallback when the requested environment has no block
- Backslash `\` continues a shell line
- `#` prefix: is a line-level comment for that line only
- `@` prefix: suppress echo of the command (show output only)
- `-` prefix: ignore non-zero exit codes and proceed executing
- `-@` prefix: both

## Full syntax example

```
# Unscoped variables
APP_PORT=8080
SERVICE_NAME=my-service

# Env-scoped variables
prod_HOST=prod.internal
staging_HOST=staging.internal

# Tail application logs
tail-logs:
    local:
        docker compose logs app --follow --tail 100
    default:
        ssh $(SSH_USER)@$(HOST) "tail -f /var/log/$(SERVICE_NAME)/app.log"

# Restart the service
restart:
    local:
        docker compose restart app
    default:
        ssh $(SSH_USER)@$(HOST) "sudo systemctl restart $(SERVICE_NAME)"
```

---

# Platform templates

Use the matching template as a starting point. Substitute real values detected from the repo wherever possible; mark unknowns with `# TODO:` comments.

## Docker Compose (local)

```
COMPOSE_FILE=docker-compose.yml
APP_CONTAINER=app
DB_CONTAINER=postgres
APP_PORT=8080

# Tail application logs
tail-logs:
    default:
        docker compose -f $(COMPOSE_FILE) logs $(APP_CONTAINER) --follow --tail 100

# Restart the app container
restart:
    default:
        docker compose -f $(COMPOSE_FILE) restart $(APP_CONTAINER)
    full:
        docker compose -f $(COMPOSE_FILE) down && \
        docker compose -f $(COMPOSE_FILE) up -d

# Check health endpoint
health-check:
    default:
        curl -sf http://localhost:$(APP_PORT)/health | jq .

# Container resource usage
check-resources:
    default:
        docker stats --no-stream $(APP_CONTAINER)

# Check downstream dependency connectivity
downstream-deps:
    default:
        docker exec $(DB_CONTAINER) pg_isready
```

## AWS ECS

```
prod_AWS_REGION=us-east-1          # TODO: set your regions
staging_AWS_REGION=us-west-2

prod_CLUSTER=my-service-prod       # TODO: set your cluster names
staging_CLUSTER=my-service-staging

prod_LOG_GROUP=/aws/ecs/my-service-prod    # TODO: set your log groups
staging_LOG_GROUP=/aws/ecs/my-service-staging

# Tail CloudWatch logs
tail-logs:
    default:
        aws logs tail $(LOG_GROUP) \
            --follow \
            --since 15m \
            --region $(AWS_REGION)

# Running vs desired task count
instance-count:
    default:
        aws ecs describe-services \
            --cluster $(CLUSTER) \
            --services my-service \
            --region $(AWS_REGION) \
            --query "services[0].{Running:runningCount,Desired:desiredCount,Pending:pendingCount}"

# Active CloudWatch alarms
check-alarms:
    default:
        aws cloudwatch describe-alarms \
            --state-value ALARM \
            --region $(AWS_REGION) \
            --query "MetricAlarms[*].{Name:AlarmName,Reason:StateReason}" \
            --output table

# Count ERROR log events in last 30 minutes
error-rate:
    default:
        aws logs filter-log-events \
            --log-group-name $(LOG_GROUP) \
            --start-time $(shell date -d '30 minutes ago' +%s000) \
            --filter-pattern "ERROR" \
            --region $(AWS_REGION) \
            --query "length(events)" \
            --output text

# Force a new ECS deployment (rollback)
rollback:
    default:
        aws ecs update-service \
            --cluster $(CLUSTER) \
            --service my-service \
            --force-new-deployment \
            --region $(AWS_REGION)
```

## Kubernetes

```
prod_NAMESPACE=my-service-prod     # TODO: set your namespaces
staging_NAMESPACE=my-service-staging
local_NAMESPACE=my-service-local

prod_CONTEXT=my-prod-context       # TODO: set your kubectl contexts
staging_CONTEXT=my-staging-context

DEPLOYMENT=my-service
CONTAINER=app

# Pod status and count
pod-count:
    default:
        kubectl get pods \
            --namespace $(NAMESPACE) \
            --context $(CONTEXT) \
            --output wide
    local:
        kubectl get pods --namespace $(NAMESPACE)

# CPU and memory usage per pod
check-resources:
    default:
        kubectl top pods \
            --namespace $(NAMESPACE) \
            --context $(CONTEXT)
    local:
        kubectl top pods --namespace $(NAMESPACE)

# Tail pod logs
tail-logs:
    default:
        kubectl logs \
            --namespace $(NAMESPACE) \
            --context $(CONTEXT) \
            --selector app=$(DEPLOYMENT) \
            --container $(CONTAINER) \
            --follow \
            --tail 100
    local:
        kubectl logs \
            --namespace $(NAMESPACE) \
            --selector app=$(DEPLOYMENT) \
            --follow \
            --tail 100

# Rollout restart
restart:
    default:
        kubectl rollout restart deployment/$(DEPLOYMENT) \
            --namespace $(NAMESPACE) \
            --context $(CONTEXT)
    local:
        kubectl rollout restart deployment/$(DEPLOYMENT) --namespace $(NAMESPACE)

# Roll back to previous revision
rollback:
    default:
        kubectl rollout undo deployment/$(DEPLOYMENT) \
            --namespace $(NAMESPACE) \
            --context $(CONTEXT)

# Deployment health
health-check:
    default:
        kubectl get deployment $(DEPLOYMENT) \
            --namespace $(NAMESPACE) \
            --context $(CONTEXT) \
            --output jsonpath='{.status.conditions[*].message}'
```

## GCP Cloud Run / GKE

```
prod_PROJECT=my-service-prod-123   # TODO: set your GCP project IDs
staging_PROJECT=my-service-staging-456

prod_REGION=us-central1            # TODO: set your regions
staging_REGION=us-east1

SERVICE=my-service
LOG_FILTER=resource.type="cloud_run_revision" AND resource.labels.service_name="my-service"

# Tail Cloud Logging
tail-logs:
    default:
        gcloud logging read '$(LOG_FILTER)' \
            --project=$(PROJECT) \
            --limit=100 \
            --format="value(timestamp,textPayload)" \
            --freshness=15m

# Cloud Run replica count
instance-count:
    default:
        gcloud run services describe $(SERVICE) \
            --project=$(PROJECT) \
            --region=$(REGION) \
            --format="value(status.observedGeneration,status.conditions[0].status)"

# Error count in last 30 minutes
error-rate:
    default:
        gcloud logging read '$(LOG_FILTER) AND severity>=ERROR' \
            --project=$(PROJECT) \
            --freshness=30m \
            --format="value(timestamp,textPayload)" | wc -l

# Roll back to previous revision
rollback:
    default:
        gcloud run services update-traffic $(SERVICE) \
            --project=$(PROJECT) \
            --region=$(REGION) \
            --to-revisions=$(shell gcloud run revisions list --service=$(SERVICE) --project=$(PROJECT) --region=$(REGION) --format="value(name)" --limit=2 | tail -1)=100
```

## Azure Container Apps / AKS

```
# AZURE_SUBSCRIPTION_ID is a secret — inject via env-file:
#   ops -e .ops_secrets.env prod tail-logs
# .ops_secrets.env:
#   prod_AZURE_SUBSCRIPTION_ID=aaaa-bbbb-cccc
#   staging_AZURE_SUBSCRIPTION_ID=dddd-eeee-ffff

prod_RESOURCE_GROUP=my-service-prod-rg     # TODO: set your resource groups
staging_RESOURCE_GROUP=my-service-staging-rg

prod_APP_NAME=my-service-prod              # TODO: set your app names
staging_APP_NAME=my-service-staging

LOG_WORKSPACE=my-service-logs

# Tail container logs via Log Analytics
tail-logs:
    default:
        az monitor log-analytics query \
            --workspace $(LOG_WORKSPACE) \
            --analytics-query "ContainerLog | order by TimeGenerated desc | take 100" \
            --subscription $(AZURE_SUBSCRIPTION_ID)

# List running replicas
instance-count:
    default:
        az containerapp replica list \
            --name $(APP_NAME) \
            --resource-group $(RESOURCE_GROUP) \
            --subscription $(AZURE_SUBSCRIPTION_ID) \
            --output table

# Active metric alerts
check-alarms:
    default:
        az monitor metrics alert list \
            --resource-group $(RESOURCE_GROUP) \
            --subscription $(AZURE_SUBSCRIPTION_ID) \
            --output table

# Error count from logs
error-rate:
    default:
        az monitor log-analytics query \
            --workspace $(LOG_WORKSPACE) \
            --analytics-query "ContainerLog | where LogEntry contains 'ERROR' | where TimeGenerated > ago(30m) | summarize count()" \
            --subscription $(AZURE_SUBSCRIPTION_ID) \
            --output table
```

## Bare-metal / SSH

```
prod_HOST=prod-srv-01.internal     # TODO: set your hostnames
staging_HOST=staging-srv-01.internal

prod_SSH_USER=deploy
staging_SSH_USER=deploy

SERVICE_NAME=my-service
LOG_FILE=/var/log/my-service/app.log

# SSH_KEY is a secret — inject via env-file:
#   ops -e .ops_secrets.env prod tail-logs
# .ops_secrets.env:
#   SSH_KEY=~/.ssh/prod_deploy_key

# Tail application log file
tail-logs:
    default:
        ssh $(SSH_USER)@$(HOST) "tail -f $(LOG_FILE)"
    prod:
        ssh -i $(SSH_KEY) $(SSH_USER)@$(HOST) "tail -f $(LOG_FILE)"

# CPU, memory, and disk usage
check-resources:
    default:
        ssh $(SSH_USER)@$(HOST) \
            "echo '=== CPU ===' && top -bn1 | grep 'Cpu(s)' && \
             echo '=== Memory ===' && free -h && \
             echo '=== Disk ===' && df -h"

# Restart the systemd service
restart:
    default:
        ssh $(SSH_USER)@$(HOST) "sudo systemctl restart $(SERVICE_NAME)"
    prod:
        ssh -i $(SSH_KEY) $(SSH_USER)@$(HOST) \
            "sudo systemctl restart $(SERVICE_NAME) && \
             sudo systemctl status $(SERVICE_NAME)"

# Service health via systemd
health-check:
    default:
        ssh $(SSH_USER)@$(HOST) "sudo systemctl status $(SERVICE_NAME) --no-pager"
```
