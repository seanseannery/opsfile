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
4. Create the agent team: use TeamCreate with a descriptive team name based on the issue. All agents MUST use the same worktree for this session.

### Step 2: Create initial tasks

Create tasks using TaskCreate for the SDLC phases:

1. **Design task** — Performed inline by the current session (team lead) acting as an `architect` subagent. Research the codebase and design the architecture for this issue. Author a design doc in `./docs/` using the template `./docs/templates/feature-doc.md`. Commit and push the doc to the feature branch but do not summarize for end user. Present the design for approval.
2. **QA design review task** — assigned to `qa`: Review proposed design doc, and provide feedback on potential quality or ux pain points with the architecture. Write a test case document in `./docs/testplans` following the template `./docs/templates/test-plan.md`
3. **Iterate on Design doc task** — After `qa` review is complete, iterate on design doc with any QA feedback.
4. **User feedback** — Present design doc to the user without a summary, just a link to the doc,  and ask for feedback/updates before beginning implementation.
5. **Implementation tasks** — assigned to engineers (spawned after user approves design in Task #4): MUST NOT start implementation before user approval. Implement the feature according to the approved design doc. Split work logically (e.g., core logic vs CLI wiring, or by component). Run `make lint` and `make test` before marking complete. Commit and push all changes to the feature branch before reporting done.
6. **QA signoff task** — assigned to `qa` after implementation is complete: Pull latest from feature branch, review all code changes, write additional tests for edge cases, run full test suite, validate behavior against the design doc requirements. Report any issues found.

Set up dependencies: implementation tasks are blocked by Task #4 (user approval). QA signoff is blocked by implementation tasks.

### Step 3: Spawn the initial team (QA only)

Spawn only QA upfront. Engineers are spawned later after the user approves the design (Step 4).

1. **qa** — `subagent_type: "qa"`, `isolation: "worktree"`, `run_in_background: true`. Tell them:
   - Their task IDs (#2 for design review, #6 for signoff)
   - **Do not read files or explore the codebase until you receive a message that a task is ready for you.** Do not poll TaskList on your own — wait for a message from team-lead.
   - For design review (Task #2): pull the latest from the feature branch before reading the design doc.
   - For signoff (Task #6): pull latest from the feature branch before reviewing code.

### Step 3.5: Complete the design, then spawn engineers

After QA review (Task #2) and design iteration (Task #3) are complete and the **user has approved the design (Task #4)**:

Spawn N engineers based on the user's earlier "How many engineers?" response. Default `subagent_type` is `backend-engineer` unless otherwise specified earlier.

  **If N=1:** Spawn the single engineer, they work in the same worktree as qa and teamlead —  no separate commit/push step needed.

  **[type]-eng-[1]** — `subagent_type: "backend-engineer"`, `run_in_background: true` (no isolation if N=1). Tell them:
  - Their task ID (#5 or whichever implementation task)
  - The feature branch name
  - The design doc path and test plan path
  - **Do not read files or explore the codebase until ready to implement.** Do not poll — start working immediately since design is already approved.
  - If using a worktree (N>1): commit and push all changes to the feature branch before marking the task complete.

  **[type]-eng-[n]** — `isolation: "worktree"`, same instructions as above but for the next parallelizable implementation task.

### Step 4: Coordinate

- After completing the design inline, commit and push the design doc to the feature branch, then notify QA to begin Task #2
- When QA design review is complete and design iteration (Task #3) is done, present to user for approval (Task #4)
- After user approves, spawn engineers (Step 3.5) and notify them to begin
- When implementation tasks complete: **before notifying QA**, verify the feature branch has the implementation commits by running `git log --oneline origin/<feature-branch> | head -5`. If commits are missing, ask the engineer to push before proceeding.
- When branch is confirmed up to date, notify QA to begin signoff (Task #6)
- When QA completes signoff, create a pull request on GitHub using the `.github/pull_request_template.md` structure
- Shut down all agents when work is complete. Switch to the main worktree and clean up any unused worktrees.

### Key rules

- The team lead performs design work inline (no architect sub-agent). This avoids worktree copy overhead for docs-only work.
- All agents modifying code use `isolation: "worktree"` on the SAME feature branch and worktree
- Engineers are spawned only after user approves the design — not upfront
- Implementation changes must be committed and pushed to the feature branch before QA signoff begins
- If any agent gets stuck or has questions, surface them to the user
- Frontend engineers should only be assigned work that is website or graphical user interface or updating user READMEs.