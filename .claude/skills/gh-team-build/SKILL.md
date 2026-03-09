---
name: gh-team-build
description: List open GitHub issues for the current repo, or load a specific issue into context by number. Use with no arguments to list the top 20 open issues. Use with an integer issue number to retrieve full details of that issue for development context.
allowed-tools: Bash(gh issue list:*), Bash(gh issue view:*), AskUserQuestion, TeamCreate, TaskCreate, TaskUpdate, TaskList, TaskGet, SendMessage, Agent
disable-model-invocation: true
user-invocable: true
---

## Arguments

$ARGUMENTS

## Your task

Check the Arguments section above and do exactly one of the following:

**Case 1 — Arguments is empty:**
Run `gh issue list --limit 20 --state open` and display the results as a clean numbered list showing each issue's number and title. Do nothing else.

**Case 2 — Arguments is an integer (issue number):**
Run `gh issue view $ARGUMENTS --comments` to retrieve the full issue details. Display the complete output including title, status, labels, assignees, body, and all comments. Treat this as active context: summarize what the issue is asking for and note any relevant technical details that would inform implementation or investigation work. Then ask the user: **"Would you like to spin up a team to implement this issue?"**

  **Case 2.a - The user says no to creating a team**
  Stop and await further instruction.

  **Case 2.b - The user says yes to creating a team**
  First, remind them **"This skill works best in tmux mode with claude-code team agents enabled"**. 
  Then ask the user **"How many engineers would you like working on it?"**
  Finally, follow the SDLC Team Workflow below using the users guidance on how many engineer agents to create for implementation steps. 

---

## SDLC Team Workflow

When the user approves spinning up a team, follow this software development lifecycle:

### Step 1: Setup

1. Determine a short branch name from the issue (following CONTRIBUTING.md conventions, e.g. `feat/issue-title` or `fix/issue-42`)
2. Create a feature branch off `main`: `git checkout -b <branch-name> main`
3. Push the branch so worktrees can use it: `git push -u origin <branch-name>`
4. Create the agent team: use TeamCreate with a descriptive team name based on the issue

### Step 2: Create initial tasks

Create tasks using TaskCreate for the SDLC phases:

1. **Design task** — Current session assumes role of `architect` (`subagent_type: "architect"`, `isolation: "worktree"`): Research the codebase and design the architecture for this issue. Author a design doc in `./docs/` using the template `./docs/templates/feature-doc.md`. Reference the issue details. Present the design for approval.
2. **QA design review task** — assigned to `qa`: Review completed design doc, and provide feedback on potential quality or ux issues with the architecture. Write a test case document in `./docs/testcases` following the template `./docs/templates/test-plan.md`
3. **Iterate on Design doc task** - After `qa` review is complete,  Iterate on design doc with any QA feedback. 
4. **User feedback** - Present design doc to user with summary and ask for feedback/updates before implementation.
5. **Implementation tasks** — assigned to `backend-1` and `backend-2`: MUST NOT start implementation before user approval. Implement the feature according to the approved design doc. Split work logically (e.g., core logic vs CLI wiring, or by component). Run `make lint` and `make test` before marking complete.
6. **QA signoff task** — assigned to `qa` after implementation is complete: Review all code changes, write additional tests for edge cases, run full test suite, validate behavior against the design doc requirements. Report any issues found.

Set up dependencies: implementation tasks are blocked by the design task. QA task is blocked by implementation tasks.

### Step 3: Spawn the team

Spawn agents using the Agent tool. All agents that modify code or docs MUST use `isolation: "worktree"` so they work in separate worktrees on the same feature branch:

1. **qa** — `subagent_type: "qa"`, `isolation: "worktree"`. Instruct to wait for design task to complete,implementation tasks to complete, then review all changes, run tests in testplan, and validate. Tell them their task ID.

2. Spin up N additional engineers based on users previous "How many engineers?" response. The subagent_type should default to backend-engineer unless otherwise specified

  **[type]-eng-[1]** - `subagent_type: "backend-engineer"`, `isolation: "worktree"`. Instruct to wait for user approval of design task.  Provide the design doc and test plan, then claim and work on their implementation task. Tell them their task ID and to check TaskList for when the design is approved and unblocked.

  **[type]-eng-[n]** — `subagent_type: "[type]-engineer"`, `isolation: "worktree"`. Same instructions as the first engineer but for the next parallelizable implementation task. Tell them their task ID.

All agents should be spawned with `run_in_background: true`.

### Step 4: Coordinate

- When initial design task is complete, notify QA to review
- When QA has completed design review and the design update is complete, notify user for feedback and signoff
- When implementation tasks complete, notify QA to begin
- When QA completes, report final status to the user and create a pull request on github
- Shut down all agents when work is complete

### Key rules

- All agents modifying code/docs use `isolation: "worktree"` on the SAME feature branch
- The architect uses `mode: "plan"` — their design MUST be approved by user and `qa` before implementation begins
- If any agent gets stuck or has questions, surface them to the user
- frontend engineers should only be assigned work that is website or graphical user interface related.