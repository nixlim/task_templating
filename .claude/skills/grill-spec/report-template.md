# Adversarial Review Report Template

Use this template when assembling the findings report in Phase 5.

The report has two finding representations:

1. The **YAML `## Issues` block** is the canonical, machine-parseable list.
   `/plan-spec --revise` consumes this block directly.
2. The **human-readable findings sections** restate the same findings for the
   spec author. They MUST stay in sync with the YAML block.

---

```markdown
# Adversarial Review: [Spec Name]

**Spec reviewed**: [path/to/spec-file.md]
**Review date**: [YYYY-MM-DD]
**Cycle**: [1 | 2 | 3]
**Verdict**: [BLOCK | REVISE | PASS]

## Executive Summary

[2-3 sentences. State counts by severity and the overall verdict. Be direct —
no softening language.]

| Severity | Count |
|----------|-------|
| BLOCKER | N |
| WARNING | N |
| INFO | N |
| **Total** | **N** |

---

## Findings

### BLOCKER Findings

#### [G-001] [Short title]

- **Dimension**: [ambiguity | incompleteness | inconsistency | infeasibility | insecurity | inoperability | incorrectness | overcomplexity | scope_alignment | codebase_fit | quality_gate | structural]
- **Affected**: [FR ID, section heading, scenario name, or quoted phrase]
- **Description**: [What is wrong. Be specific. Reference the exact text.]
- **Impact**: [What happens if this ships as-is. Concrete scenario.]
- **Fix hint**: [Exactly what to change. Provide rewritten text where possible.]

---

#### [G-002] [Short title]

[Same structure as above]

---

### WARNING Findings

#### [G-003] [Short title]

- **Dimension**: [dimension]
- **Affected**: [reference]
- **Description**: [What is wrong]
- **Fix hint**: [Specific fix]

---

### INFO Findings

#### [G-NNN] [Short title]

- **Dimension**: [dimension]
- **Affected**: [reference]
- **Suggestion**: [Improvement idea or deferred-reason prompt]

---

## Structural / Narrative Integrity

A single adaptive section. Each row applies if its precondition is met by the
detected input shape; rows whose precondition does not apply are omitted from
the rendered table (do not write "N/A" rows).

| Check | Applies when | Result | Notes |
|-------|--------------|--------|-------|
| Every FR-NNN has at least one behavioural scenario | Spec contains FR-NNN IDs | PASS/FAIL | [Details if FAIL] |
| Every behavioural scenario traces back to an FR-NNN | Spec contains `## Behavioral Scenarios` | PASS/FAIL | [Details if FAIL] |
| Acceptance criteria are falsifiable and measurable | Spec contains acceptance criteria | PASS/FAIL | [Details if FAIL] |
| Cross-references resolve (no dangling IDs) | Spec uses any ID scheme (FR-, AC-, SC-) | PASS/FAIL | [Details if FAIL] |
| Scope boundaries are explicit (in / out / deferred) | Always | PASS/FAIL | [Details if FAIL] |
| Error and failure modes addressed | Always | PASS/FAIL | [Details if FAIL] |
| Dependencies between requirements identified | Spec has multiple FRs / sections | PASS/FAIL | [Details if FAIL] |
| Actors (users, systems, services, jobs) named | Always | PASS/FAIL | [Details if FAIL] |
| Implementation detail sufficient to begin work | Always | PASS/FAIL | [Details if FAIL] |
| Assumptions and constraints stated explicitly | Always | PASS/FAIL | [Details if FAIL] |

---

## Alignment Summary

### Scope Alignment (Check 9)

[One paragraph. State whether requirements trace back to the originating user
need / ticket / problem statement, and call out any drift. Reference finding
IDs where applicable.]

### Codebase Fit (Check 10)

[One paragraph. State whether the spec respects existing naming, module
boundaries, and architectural idioms, and call out any mismatches. Reference
finding IDs where applicable.]

---

## Quality Gate Summary

List which Tier 3 anti-bloat checks fired and where. Each row maps to one or
more findings above.

| Check | Triggered? | Locations |
|-------|-----------|-----------|
| QG-USER-STORY (user-story narrative) | Yes/No | [Section / phrase] |
| QG-RESURRECTED (deleted section reappeared) | Yes/No | [Section name] |
| QG-TEST-CATALOG (numbered test catalog >10) | Yes/No | [Section] |
| QG-WEASEL (weasel words in FRs) | Yes/No | [FR IDs + words] |
| QG-FALSIFY (non-falsifiable AC) | Yes/No | [AC reference] |
| QG-SPEC-FAIL (specificity failure) | Yes/No | [Requirement] |
| QG-DEFER-NO-REASON (deferred without reason) | Yes/No | [Item] |

---

## Test Coverage Assessment

Missing test categories only — no Dataset Gaps table, no per-scenario list.

| Missing category | Affected scenarios / FRs | Why it matters |
|------------------|--------------------------|----------------|
| [e.g. concurrency] | [FR-007, FR-009] | [One sentence on the risk] |
| [e.g. negative-path tests] | [FR-012] | [...] |

---

## Unasked Questions

These are questions the spec should have answered but did not. The spec
author should address each one before proceeding to implementation.

1. [Question about missing requirement or undecided design choice]
2. [Question about unclear failure handling]
3. [Question about missing integration concern]
4. [...]

---

## Verdict Rationale

[1-2 paragraphs explaining the verdict. Reference the most impactful findings
by ID. State clearly what must change before implementation can proceed. If
this is cycle 3 or a stalled cycle, say so explicitly.]

### Recommended Next Actions

- [ ] [Specific action item — reference finding ID]
- [ ] [Specific action item — reference finding ID]
- [ ] [Specific action item — reference finding ID]

---

## Issues

The canonical machine-parseable list. `/plan-spec --revise` reads this block.
Every finding above MUST appear here, and vice versa. Use sequential IDs
starting at `G-001`.

```yaml
issues:
  - id: G-001
    dimension: ambiguity        # ambiguity | incompleteness | inconsistency | infeasibility | insecurity | inoperability | incorrectness | overcomplexity | scope_alignment | codebase_fit | quality_gate | structural
    severity: blocker           # blocker | warning | info
    affected: "FR-007"          # FR ID, section heading, or quoted phrase
    description: |
      One-paragraph problem statement. Reference the exact text or section.
    fix_hint: |
      Concrete, actionable next step. Rewritten phrasing where possible.

  - id: G-002
    dimension: scope_alignment
    severity: warning
    affected: "FR-012"
    description: |
      ...
    fix_hint: |
      ...

  # ... continue for every finding
```
```
