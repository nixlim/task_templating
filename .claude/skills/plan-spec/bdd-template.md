# BDD Scenario Reference

Format and rules for Given-When-Then scenarios in `/plan-spec` outputs. The
goal is short scenarios that trace cleanly to functional requirements and
fall into exactly one of three categories.

## Format

```
### Scenario: [Descriptive, action-oriented title]

**Traces to**: FR-NNN[, FR-NNN ...]
**Category**: Happy Path | Error Path | Edge Case

- **Given** [a precondition that establishes the starting state]
- **And** [an additional precondition, if needed]
- **When** [a single action performed by the actor]
- **Then** [an observable, verifiable outcome]
- **And** [an additional assertion, if needed]
- **But** [something that should NOT happen, if relevant]
```

## Mandatory Rules

1. **Trace to FR-IDs.** Every scenario MUST have a `Traces to:` line listing
   one or more FR-IDs from the spec's Functional Requirements section. No
   user-story tracing — this format does not have user stories.

2. **One of three categories.** Every scenario MUST have exactly one
   `Category:` value:
   - **Happy Path**: the main success flow with expected inputs.
   - **Error Path**: the actor or a dependency triggers an error; the
     scenario verifies correct handling, messaging, and recovery.
   - **Edge Case**: boundary values, unusual inputs, race conditions, or
     rare-but-possible situations.

   No "Alternate Path" — if a flow is valid and succeeds, it is a Happy
   Path scenario for whichever FR governs that branch.

3. **One action per When.** The When step describes exactly one action.
   Compound actions belong in separate scenarios.

4. **Observable assertions only.** Then steps assert externally visible
   outcomes (HTTP responses, persisted state, emitted events, file
   contents, UI text). Never assert internal implementation details.

5. **Concrete preconditions.** Given steps describe specific, reproducible
   state — never "the system is ready" or "the user exists" without
   identifying the user.

## Anti-Patterns

1. **Implementation leakage.** "Given a `users` table with columns id, name,
   email" couples the scenario to schema. Use: "Given a registered user
   with email alice@example.com."

2. **Compound When.** "When the user logs in and navigates to settings and
   changes their password" is three scenarios, not one.

3. **Vague assertion.** "Then it works" is not a Then. State the observable
   outcome.

4. **Orphan scenario.** A scenario without `Traces to:` is disconnected from
   requirements. If you cannot map it to an FR, either add the FR or drop
   the scenario.

## Naming

- Use the actor's perspective: "User resets password with valid token", not
  "Password reset token validation."
- Include the differentiator: "User submits form with empty required field",
  not just "Form validation."
- For error paths, name the error: "User receives rate-limit error after
  10 requests in 1 minute."

## Coverage

Per the SKILL.md coverage rule:

- Every MUST FR has at least one Happy Path scenario.
- Every FR involving external input, dependencies, or failure modes has at
  least one Error Path scenario.
- Boundary conditions identified during discovery get an Edge Case scenario.
