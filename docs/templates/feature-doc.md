# Feature: [Feature Name]


## 1. Problem Statement & High-Level Goals

### Problem
[Describe the problem this feature solves. What pain point exists today? Who is affected?]
[Reference related github issue]

### Goals
- [ ] [Primary goal]
- [ ] [Secondary goal]
- [ ] [Tertiary goal]

### Non-Goals
- [Explicitly list what this feature will NOT do]

---

## 2. Functional Requirements

### FR-1: [Requirement Name]
[Description of the requirement and expected behavior]

### FR-2: [Requirement Name]
[Description of the requirement and expected behavior]

### FR-3: [Requirement Name]
[Description of the requirement and expected behavior]

### Example Usage

[example user scenario]
[examples of relevant bash cli call and output]
[example of relevant Opsfile configuration]

---

## 3. Non-Functional Requirements

| ID | Category | Requirement | Notes |
|----|----------|-------------|-------|
| NFR-1 | Performance | [e.g., < 100ms response time] | |
| NFR-2 | Compatibility | [e.g., Linux, macOS, Windows] | |
| NFR-3 | Reliability | [e.g., graceful error handling] | |
| NFR-4 | Security | [e.g., no credential leakage] | |
| NFR-5 | Maintainability | [e.g., test coverage >= existing] | |

---

## 4. Architecture & Implementation Proposal

### Overview
[High-level description of the proposed implementation approach]

### Component Design
[Describe new or modified components and their responsibilities]

### Data Flow ()
```
[Step 1] -> [Step 2] -> [Step 3] -> [Output]
```

#### Sequence Diagram
[Insert a UML sequence diagram explaing the flow]

### Key Design Decisions
- **[Decision 1]:** [Rationale]
- **[Decision 2]:** [Rationale]

### Files to Create / Modify

| File | Action | Description |
|------|--------|-------------|
| `**/[file].go` | Create | [What it does] |
| `**/[file].go` | Modify | [What changes] |
| `**/[file]_test.go` | Create | [Test coverage] |

---

## 5. Alternatives Considered

### Alternative A: [Name]

**Description:** [How this approach would work]

**Pros:**
- [Advantage 1]
- [Advantage 2]

**Cons:**
- [Disadvantage 1]
- [Disadvantage 2]

**Why not chosen:** [Brief explanation]

---

### Alternative B: [Name]

**Description:** [How this approach would work]

**Pros:**
- [Advantage 1]
- [Advantage 2]

**Cons:**
- [Disadvantage 1]
- [Disadvantage 2]

**Why not chosen:** [Brief explanation]

---

## Open Questions
- [ ] [Any unresolved questions or decisions needed]

---

## 6. Task Breakdown

### Phase 1: Foundation
- [ ] [Task 1 -- e.g., define types/interfaces]
- [ ] [Task 2 -- e.g., implement core logic]
- [ ] [Task 3 -- e.g., write unit tests for core logic]

### Phase 2: Integration
- [ ] [Task 4 -- e.g., wire into CLI entry point]
- [ ] [Task 5 -- e.g., add flag/argument parsing support]
- [ ] [Task 6 -- e.g., write integration tests]

### Phase 3: Polish
- [ ] [Task 7 -- e.g., error messages and edge cases]
- [ ] [Task 8 -- e.g., update README / user docs]
- [ ] [Task 9 -- e.g., add example Opsfile if applicable]

---


