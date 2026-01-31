# Analysis of lambda-LLM as an Intermediate Representation for AI Coding Agents

**Version:** 0.1.0
**Status:** Draft
**Date:** 2026-01-31

---

## 1. Executive Summary

lambda-LLM is a proposed formal intermediate language that sits between human intent and generated code. It combines S-expression syntax, refinement types, an effect/capability system, and SMT-based verification to create a "validation gateway" for AI-generated logic. The concept has genuine merit for specific domains (financial calculations, authorization logic, state machines) where formal correctness guarantees justify the implementation cost. However, as a general-purpose task specification format for AI coding agents, it is over-engineered — the translation step from English to lambda-LLM is itself the weakest link in the pipeline, and the majority of software engineering tasks involve concerns (UI layout, API design, concurrency patterns, file organization) that do not reduce to verifiable mathematical assertions.

This document provides a detailed analysis of lambda-LLM's strengths and weaknesses, compares it to the Structured Task Template approach, and recommends where each should be applied.

---

## 2. lambda-LLM Overview

### 2.1 Core Concepts

lambda-LLM programs are collections of **Nodes** — content-addressed, self-contained units of logic. Each node declares:

- **Dependencies** (other nodes or libraries it requires)
- **Effects** (capabilities it needs: database access, network, clock)
- **Contracts** (typed inputs, typed outputs, invariants)
- **Logic** (the computation itself, in S-expression form)
- **Checks** (example-based test cases)

### 2.2 Key Technical Features

| Feature | Description |
|---|---|
| **Refinement Types** | Types carry constraints: `f64(0..1)` instead of bare `f64`. Narrows the valid domain at the type level. |
| **Effect System** | Capability-based security. A node cannot perform I/O unless it declares the capability (e.g., `DB.Write`, `Http.Out("api.stripe.com")`). |
| **SMT Verification** | Invariants like `new_balance >= 0` are checked by a Z3 solver against the logic. The solver either proves the invariant holds for all inputs or produces a counterexample. |
| **Algebraic Data Types** | Union types model domain concepts precisely: `Union(Fixed: f64, Pct: f64(0..1))` instead of stringly-typed alternatives. |
| **Content Addressing** | Nodes are identified by hash, enabling caching and deduplication. |

### 2.3 The Pipeline

```
Step 1: Human writes requirements (English)
Step 2: LLM synthesizes lambda-LLM code from requirements
Step 3: Gatekeeper verifies the lambda-LLM code:
        a) Parser     — S-expression syntax valid?
        b) Type Checker — types used consistently?
        c) SMT Solver  — invariants hold for all inputs?
        d) Effect Checker — no undeclared side effects?
Step 4: If all pass, transpile to target language (Go, Rust, etc.)
Step 5: If rejected, structured error returned to LLM for retry
```

### 2.4 Error Feedback Loop

When the Gatekeeper rejects code, it returns a structured error:

```json
{
  "status": "REJECTED",
  "reason": "INVARIANT_VIOLATION",
  "details": {
    "node": "Finance.ProcessRefund",
    "assertion": "new_balance >= 0",
    "counter_example": {
      "t.balance": 50.0,
      "amount": 60.0,
      "result": -10.0
    }
  },
  "instruction": "Add a check to verify 'amount <= t.balance' before proceeding."
}
```

This is one of lambda-LLM's strongest features: the error is specific, actionable, and machine-readable. The LLM can use it to self-correct without human intervention.

---

## 3. Strengths

### 3.1 Formal Verification Catches Edge Cases Humans Miss

The SMT solver reasons about *all possible inputs*, not just the test cases the developer thought to write. For financial logic, this is transformative:

- "Can the balance ever go negative?" becomes a provable assertion, not a hope.
- "Can the discount ever exceed the price?" gets a counterexample if the logic doesn't prevent it.
- Off-by-one errors, boundary conditions, and floating-point edge cases are caught at specification time, before any target code is generated.

**Significance: HIGH for mathematical/financial domains.**

### 3.2 Effect System Enforces Capability-Based Security

Declaring effects explicitly (`Http.Out("api.stripe.com")`) means:

- A node that processes payments cannot secretly exfiltrate data to an unauthorized endpoint.
- A "pure calculation" node is provably free of side effects.
- An agent cannot accidentally introduce database writes into a read-only operation.

This is genuinely useful for security-critical systems where the blast radius of an LLM hallucination must be contained.

**Significance: HIGH for security-critical systems.**

### 3.3 Refinement Types Make Impossible States Unrepresentable

`f64(0..1)` for a percentage is more informative than a comment saying "must be between 0 and 1." The type system enforces it. This eliminates an entire class of "the value was supposed to be in range X but the caller passed Y" bugs.

**Significance: MEDIUM — achievable with runtime validation, but lambda-LLM catches it earlier.**

### 3.4 Machine-Readable Errors Enable Self-Correction Loops

The structured error format means an LLM can:

1. Receive the counterexample.
2. Understand exactly which invariant failed and why.
3. Generate a corrected version.
4. Resubmit for verification.

This closed loop can iterate automatically without human involvement, potentially converging on correct code faster than human code review.

**Significance: HIGH — this is a genuinely novel contribution.**

### 3.5 Content Addressing Enables Caching and Reuse

If a node is content-addressed (identified by hash of its definition), then:

- Identical logic across projects is recognized and reused.
- Previously verified nodes don't need re-verification.
- A "standard library" of verified nodes accumulates over time.

**Significance: MEDIUM — useful at scale, negligible for small projects.**

---

## 4. Weaknesses

### 4.1 The Translation Problem (Critical)

The most error-prone step is **Step 2: LLM synthesizes lambda-LLM from English**. This is where ambiguity lives. The Gatekeeper only verifies internal consistency — it cannot detect requirements misunderstanding.

**Example:**

Human says: "Process a refund."
LLM generates lambda-LLM that processes the refund by crediting the *seller* instead of the *buyer*.

The types check out. The invariants hold. The effects are declared. The code is formally verified to be *wrong*.

This is the **valid-but-wrong problem**: the specification is internally consistent but does not match the human's intent. No amount of type checking or SMT solving can detect a requirements error, because the requirements themselves were lost in translation.

**Severity: CRITICAL. This undermines the core value proposition for general-purpose use.**

### 4.2 Expressiveness Ceiling

lambda-LLM's examples work well for:
- Calculate a discounted total
- Process a financial refund
- Verify a webhook signature

These are algorithmic, transactional, and have clear input/output boundaries. But the majority of software engineering tasks look like:

| Task Type | Why lambda-LLM Struggles |
|---|---|
| "Add a `--format` CLI flag" | Involves file layout, flag registration, string routing. No invariants to verify. |
| "Refactor the chunker" | Structural change across multiple files. The "logic" is architectural, not mathematical. |
| "Fix the race condition" | Concurrency semantics. S-expressions don't model goroutines, channels, or mutex ordering. |
| "Improve error messages" | Subjective quality. No formal notion of "better error message." |
| "Add logging to the pipeline" | Cross-cutting concern. Touches many nodes but has no mathematical contract. |
| "Design the database schema" | Relational modeling. lambda-LLM's type system doesn't cover SQL DDL, indexes, or migrations. |

For these tasks — which constitute the majority of real software work — lambda-LLM either degenerates into natural language wrapped in S-expressions (defeating the purpose) or simply cannot express the task at all.

**Severity: HIGH. Limits applicability to perhaps 15-25% of real software tasks.**

### 4.3 Implementation Cost

Building the lambda-LLM pipeline requires:

| Component | Estimated Effort | Dependencies |
|---|---|---|
| S-expression parser | 1-2 weeks | Standard parsing techniques |
| Type checker with refinement types | 3-6 weeks | Bidirectional type inference, subtyping |
| Z3/SMT integration | 2-4 weeks | Z3 bindings, assertion encoding, model extraction |
| Effect checker | 1-2 weeks | Capability lattice, flow analysis |
| Code generator (to Go) | 4-8 weeks | AST mapping, idiom translation, import management |
| Standard library | 2-4 weeks ongoing | DB, Crypto, Http module definitions |
| Error feedback formatter | 1 week | JSON schema, counterexample rendering |
| **Total** | **14-27 weeks** | **Assumes one experienced engineer** |

This is a substantial investment. For comparison, the Structured Task Template approach requires zero implementation — it is a documentation convention.

**Severity: HIGH. The ROI must be very large to justify this investment.**

### 4.4 Maintenance Burden

Once built, the lambda-LLM toolchain becomes a dependency of the entire development pipeline:

- **Spec evolution:** Adding new features to the language requires updating the parser, type checker, SMT encoding, and code generator in lockstep.
- **Target language changes:** A new Go version with different idioms may require codegen updates.
- **Z3 upgrades:** SMT solver updates can change performance characteristics and edge-case behavior.
- **Developer onboarding:** Every contributor must learn lambda-LLM syntax in addition to the target language.

**Severity: MEDIUM. Manageable for a dedicated team, burdensome for small projects.**

### 4.5 Two LLM Round-Trips

The pipeline requires:

1. English -> lambda-LLM (LLM call #1)
2. lambda-LLM -> Go (code generation, possibly LLM call #2 if using LLM for codegen)

Each LLM call introduces latency, cost, and error probability. The direct approach (English -> Go with structured task templates) requires one LLM interaction. Each additional step in the pipeline is a potential point of failure.

**Severity: MEDIUM. Mitigable with caching and fast verification, but never eliminable.**

---

## 5. Comparison: Structured Templates vs lambda-LLM

### 5.1 Dimension-by-Dimension Analysis

| Dimension | Structured Templates | lambda-LLM |
|---|---|---|
| **Ambiguity reduction** | High. Explicit fields force completeness. But constraints are natural language. | Very High for expressible tasks. Refinement types and invariants are formal. But the English-to-lambda-LLM step reintroduces ambiguity. |
| **Correctness guarantees** | None (documentation format). Correctness is verified by tests post-implementation. | Strong for expressed invariants. SMT proves properties for all inputs, not just test cases. |
| **Task coverage** | ~95% of real software tasks. Handles CLI features, refactoring, integration, architecture. | ~15-25% of real software tasks. Limited to algorithmic/transactional logic with clear I/O contracts. |
| **Implementation cost** | Zero. Text convention. No tooling required. | 14-27 engineer-weeks for the core pipeline. Ongoing maintenance thereafter. |
| **Learning curve** | Low. Any developer can read and write templates in minutes. | High. Requires understanding S-expressions, refinement types, effect systems, and SMT concepts. |
| **Error feedback quality** | Informal. Agent reports which acceptance criteria failed. Human interprets. | Excellent. Structured, machine-readable errors with counterexamples. Enables automated self-correction. |
| **Composability** | Via DEPENDS_ON field and milestone grouping. Simple DAG. | Via content-addressed dependencies and typed interfaces. Richer composition model. |
| **Tooling requirements** | Text editor. | Parser, type checker, SMT solver (Z3), code generator, standard library. |
| **Iteration speed** | Fast. Edit a text field, re-run the agent. | Slow. Edit lambda-LLM, re-verify, re-generate, re-test. |
| **Portability** | Language-agnostic. Works for Go, Python, Rust, JavaScript. | Requires a code generator per target language. |

### 5.2 Failure Mode Analysis

| Failure Mode | Structured Templates | lambda-LLM |
|---|---|---|
| **Requirements misunderstanding** | Possible. Mitigated by explicit GOAL, ACCEPTANCE, NON_GOALS fields. Agent halts on ambiguity. | Equally possible. Gatekeeper cannot detect requirements errors — only internal consistency. |
| **Implementation divergence** | Possible. Agent may satisfy ACCEPTANCE but violate unstated assumptions. Mitigated by CONSTRAINTS and FILES_SCOPE. | Less likely for verified properties. Impossible for SMT-proven invariants. But unverified aspects have the same risk. |
| **Over-engineering** | Low risk. NON_GOALS field and scope limits prevent creep. | Moderate risk. The formalism encourages specifying more than necessary. |
| **Under-specification** | Caught by validation checklist (V1-V10). Required fields enforce minimum completeness. | Caught by type checker and effect checker. But only for expressible properties. |
| **Toolchain failure** | N/A (no toolchain). | Parser bugs, Z3 timeouts, codegen errors. Each is a potential blocker. |

---

## 6. Suitability Matrix

### 6.1 By Task Type

| Task Type | Best Approach | Rationale |
|---|---|---|
| **Financial calculations** | lambda-LLM | Invariants (non-negative balance, correct totals) are mathematically provable. The cost of a bug is high. |
| **Authorization / permissions** | lambda-LLM | Effect system naturally models capability-based security. Formal verification prevents privilege escalation. |
| **State machines** | lambda-LLM | Transitions are enumerable. Reachability and deadlock-freedom are provable. |
| **Data validation rules** | lambda-LLM | Refinement types are a direct fit. Input validation contracts are small and verifiable. |
| **Cryptographic protocol logic** | lambda-LLM | Formal verification of protocol steps prevents subtle security flaws. |
| **CLI features** | Structured Templates | Flag registration, output formatting, and error messaging are structural, not mathematical. |
| **API endpoint implementation** | Structured Templates | Request routing, middleware, and response formatting don't have meaningful invariants. |
| **Database schema / migrations** | Structured Templates | Relational modeling is not expressible in lambda-LLM's type system. |
| **Refactoring** | Structured Templates | "Zero behavior change" is verified by existing tests, not by formal proof. |
| **UI / frontend work** | Structured Templates | Visual layout and interaction patterns have no formal specification. |
| **Integration / glue code** | Structured Templates | Wiring services together is structural, not algorithmic. |
| **Performance optimization** | Structured Templates | Performance is empirical (measured), not formal (proven). |
| **Concurrency / parallelism** | Structured Templates | Goroutine scheduling, channel semantics, and mutex ordering are beyond lambda-LLM's current model. |
| **Error handling patterns** | Structured Templates | "Good error messages" is a qualitative judgment. |
| **Documentation** | Structured Templates | Prose quality cannot be formally verified. |

### 6.2 By Project Phase

| Phase | Best Approach | Rationale |
|---|---|---|
| **Early prototyping** | Structured Templates | Speed matters more than formal correctness. Requirements are still shifting. |
| **Core algorithm development** | lambda-LLM (targeted) | Once requirements stabilize, critical algorithms benefit from formal verification. |
| **Integration / wiring** | Structured Templates | Connecting components is structural work. |
| **Hardening / security audit** | lambda-LLM (targeted) | Formal verification of security-critical paths. |
| **Maintenance / bug fixes** | Structured Templates | Most bugs are in the glue, not the math. |

---

## 7. Hybrid Approach

The two approaches are not mutually exclusive. A pragmatic architecture uses Structured Templates as the default and embeds lambda-LLM nodes for critical subsystems.

### 7.1 Architecture

```
Project Task File (Structured Templates)
  |
  +-- task: cli-export-format-flag          (Structured Template)
  +-- task: weaviate-hybrid-search          (Structured Template)
  +-- task: calculate-relevance-score       (Structured Template + lambda-LLM node)
  |     |
  |     +-- NOTES: "Core scoring logic formally verified. See lambda-LLM node below."
  |     +-- EMBEDDED_FORMAL_SPEC:
  |           [Node: Relevance.Score]
  |           {Effects: [Pure]}
  |           {Contract:
  |             (In:  (bm25: f64(0..1)) (vector: f64(0..1)) (alpha: f64(0..1)))
  |             (Out: (score: f64(0..1)))
  |             (Inv: [score == (Mul alpha vector) + (Mul (Sub 1.0 alpha) bm25)])
  |           }
  |           {Logic: (Add (Mul alpha vector) (Mul (Sub 1.0 alpha) bm25))}
  |           {Checks: [(0.8, 0.6, 0.75) -> 0.65]}
  |
  +-- task: refactor-formatter-interface    (Structured Template)
```

### 7.2 Rules for When to Embed lambda-LLM

Use a lambda-LLM node within a Structured Template when ALL of the following are true:

1. **The logic is algorithmic** — it computes a result from inputs, not orchestrates components.
2. **The invariants are expressible** — you can state a mathematical property that must hold for all inputs.
3. **The cost of a bug is high** — incorrect results cause financial loss, security breach, or data corruption.
4. **The function is small** — under ~50 lines of target code. lambda-LLM is not suited for large subsystems.
5. **You have the tooling** — a parser and Z3 integration are available (or you're willing to verify by hand).

If any condition is false, use a Structured Template alone.

### 7.3 Embedding Syntax

Within a Structured Template, add an optional `FORMAL_SPEC` field:

```yaml
FORMAL_SPEC:
  notation: lambda-llm-v0.1
  node: |
    [Node: <Name>]
    {Effects: [...]}
    {Contract: ...}
    {Logic: ...}
    {Checks: [...]}
  verification_status: unverified | verified | verified-with-caveats
  verification_notes: "<any notes on what was/wasn't proven>"
```

The agent should:
1. Implement the function to match the lambda-LLM contract.
2. Write tests corresponding to the `Checks` examples.
3. If Z3 tooling is available, verify the invariants.
4. If Z3 is unavailable, add a comment in the code: `// INVARIANT: <assertion> — verified by hand / not yet formally verified`.

---

## 8. Cost-Benefit Analysis

### 8.1 Structured Templates

| Item | Cost | Benefit |
|---|---|---|
| Design the template schema | 2-4 hours (one-time) | Reusable across all projects |
| Write a task in template format | 10-30 minutes per task | Eliminates ambiguity, reduces rework by ~60-80% (estimated) |
| Train agents to follow the protocol | Include spec in system prompt | Consistent execution across sessions |
| Maintain the spec | ~1 hour/month as patterns emerge | Spec improves with use |
| **Total first-year cost** | **~40-80 hours** | **Applicable to 95% of tasks** |

### 8.2 lambda-LLM (Full Pipeline)

| Item | Cost | Benefit |
|---|---|---|
| Build parser | 1-2 weeks | S-expression validation |
| Build type checker | 3-6 weeks | Type safety for lambda-LLM nodes |
| Integrate Z3 | 2-4 weeks | Invariant verification |
| Build effect checker | 1-2 weeks | Capability enforcement |
| Build code generator (one target) | 4-8 weeks | Automated Go/Rust output |
| Build standard library | 2-4 weeks | DB, Crypto, Http modules |
| Ongoing maintenance | ~4-8 hours/week | Keep toolchain current |
| **Total first-year cost** | **~600-1200 hours** | **Applicable to 15-25% of tasks** |

### 8.3 ROI Crossover

For lambda-LLM to justify its implementation cost, the bugs it prevents must cost more than the toolchain cost.

**Break-even scenario:** If your project has ~50 critical algorithmic functions, and each undetected bug costs $5,000-$20,000 in rework or production incidents, and lambda-LLM catches 2-5 bugs that tests would miss, the break-even is:

- Low estimate: 50 functions x 4% catch rate x $5,000 = $10,000 saved vs ~$60,000 toolchain cost. **Does not break even.**
- High estimate: 50 functions x 10% catch rate x $20,000 = $100,000 saved vs ~$60,000 toolchain cost. **Breaks even in year 1.**

The ROI is positive only for projects with a high density of critical algorithmic functions and a high cost of bugs (fintech, healthcare, aerospace). For typical software projects, the ROI is negative.

### 8.4 lambda-LLM (Lightweight / Manual)

There is a middle path: use lambda-LLM as a **documentation convention** without building the toolchain.

| Item | Cost | Benefit |
|---|---|---|
| Write lambda-LLM specs for critical functions | 20-40 minutes per function | Forces precise thinking about invariants and effects |
| Manually verify invariants (no Z3) | 10-20 minutes per function | Catches some edge cases through structured reasoning |
| Agent implements to match the contract | Same as without lambda-LLM | Contract is more precise than prose |
| **Total first-year cost** | **~20-60 hours** (for ~50 functions) | **Better than prose, cheaper than toolchain** |

This "lambda-LLM Lite" approach captures the *thinking discipline* of formal specification without the implementation cost of the verification toolchain. It combines well with the Structured Template as the `FORMAL_SPEC` field described in Section 7.3.

---

## 9. Recommendation

### 9.1 Primary Approach: Structured Task Templates

Use the Structured Task Template (as defined in `STRUCTURED_TEMPLATE_SPEC.md`) as the default task specification format for all work. It covers ~95% of real software tasks, has zero implementation cost, and provides sufficient structure to eliminate the most common ambiguity failures in AI agent execution.

### 9.2 Supplementary Approach: lambda-LLM Lite (Embedded Formal Specs)

For critical algorithmic functions — financial calculations, scoring algorithms, authorization checks, data validation rules — embed a lambda-LLM node within the Structured Template's `FORMAL_SPEC` field. Use this as a documentation and reasoning aid, not as input to an automated verification pipeline.

This gives you 80% of lambda-LLM's *thinking benefit* (forces you to state invariants, effects, and contracts precisely) at 5% of the implementation cost (no parser, no Z3, no codegen).

### 9.3 Future Consideration: Full lambda-LLM Pipeline

Invest in the full lambda-LLM toolchain only if:

1. Your project has **50+ critical algorithmic functions** where formal verification would add value.
2. The **cost of a single undetected bug** in those functions exceeds $10,000.
3. You have **dedicated tooling engineers** who can build and maintain the pipeline.
4. The pipeline would be **reused across multiple projects**, amortizing the cost.

For most software projects, including typical web applications, CLI tools, and CRUD systems, the full pipeline will not pay for itself.

### 9.4 Summary Decision Matrix

| Your Situation | Recommendation |
|---|---|
| General software project, mixed task types | Structured Templates only |
| Project with some critical algorithms | Structured Templates + lambda-LLM Lite (embedded specs) |
| Fintech / healthcare / aerospace with many critical algorithms | Structured Templates + evaluate full lambda-LLM pipeline |
| Research project exploring formal methods | Build the pipeline — the learning is the value |

---

## Appendix A: lambda-LLM Syntax Reference (for Embedded Use)

For teams using lambda-LLM Lite (Section 8.4), here is the minimal syntax needed for embedded formal specs:

```lisp
[Node: <Namespace.Name>]
{Effects: [<Pure | DB.Read | DB.Write | Network.Out("<domain>") | Filesystem.Read | Filesystem.Write | Clock.Now>]}
{Errors:  Union(<ErrorName>(<Type>), ...)}

{Contract:
  (In:  (<name>: <Type> [where <Constraint>]))
  (Out: (<name>: <Type> [where <Constraint>]))
  (Inv: [<InvariantExpression>, ...])
}

{Logic:
  (<Expression>)
}

{Checks: [(<input_values>) -> <expected_output>, ...]}
```

### Expression Forms

| Form | Meaning | Example |
|---|---|---|
| `(Add a b)` | Addition | `(Add price tax)` |
| `(Sub a b)` | Subtraction | `(Sub balance amount)` |
| `(Mul a b)` | Multiplication | `(Mul price (Sub 1.0 discount))` |
| `(Div a b)` | Division | `(Div total count)` |
| `(If cond then else)` | Conditional | `(If (Lt balance amount) (Raise InsufficientFunds) ...)` |
| `(Match val cases...)` | Pattern match | `(Match discount (Fixed v) -> ... (Pct p) -> ...)` |
| `(Case opt ...)` | Option matching | `(Case result (None) -> ... (Some v) -> ...)` |
| `(Lt a b)` | Less than | `(Lt amount balance)` |
| `(Gt a b)` | Greater than | `(Gt score threshold)` |
| `(Eq a b)` | Equality | `(Eq status "active")` |
| `(With [bindings] body)` | Let binding | `(With [x (DB.Get id)] ...)` |
| `(Do exprs...)` | Sequence | `(Do (DB.Write ...) (Log ...) result)` |
| `(Raise error)` | Raise error | `(Raise TransactionNotFound)` |

### Type Refinement Syntax

| Syntax | Meaning |
|---|---|
| `f64` | Any 64-bit float |
| `f64(> 0)` | Positive float |
| `f64(0..1)` | Float in [0, 1] |
| `i64(1..100)` | Integer in [1, 100] |
| `string(len: 1..255)` | String with length 1-255 |
| `Union(A: T1, B: T2)` | Tagged union |
| `Optional(T)` | Nullable value |

---

## Appendix B: Worked Comparison

The same task expressed in both formats, for direct comparison.

### Task: Calculate a weighted relevance score from BM25 and vector similarity scores.

#### Structured Template Version

```yaml
TASK_ID: calculate-relevance-score
TASK_NAME: Implement weighted relevance score from BM25 and vector similarity
GOAL: A pure function combines BM25 and vector similarity scores using a configurable alpha weight, returning a score in [0, 1].

INPUTS:
  - name: bm25_score
    type: f64
    constraints: 0.0 <= bm25_score <= 1.0
    source: Weaviate BM25 search result
  - name: vector_score
    type: f64
    constraints: 0.0 <= vector_score <= 1.0
    source: Weaviate vector search result
  - name: alpha
    type: f64
    constraints: 0.0 <= alpha <= 1.0
    source: Config, default 0.75

OUTPUTS:
  - name: score
    type: f64
    constraints: 0.0 <= score <= 1.0
    destination: Return value, used for result ranking

ACCEPTANCE:
  - RelevanceScore(0.8, 0.6, 0.75) == 0.65
  - RelevanceScore(1.0, 1.0, 0.5) == 1.0
  - RelevanceScore(0.0, 0.0, 0.5) == 0.0
  - RelevanceScore(0.0, 1.0, 1.0) == 1.0 (pure vector)
  - RelevanceScore(1.0, 0.0, 0.0) == 1.0 (pure BM25)
  - Output is always in [0, 1] for any valid inputs

DEPENDS_ON: N/A (pure function)

CONSTRAINTS:
  - Pure function: no side effects
  - Formula: score = alpha * vector_score + (1 - alpha) * bm25_score
  - Must not panic on any input within the declared constraints

FILES_SCOPE:
  - internal/search/scoring.go
  - internal/search/scoring_test.go

EFFECTS: None

PRIORITY: high
ESTIMATE: trivial
```

#### lambda-LLM Version

```lisp
[Node: Search.RelevanceScore]
{Effects: [Pure]}
{Errors: Union(InvalidAlpha(f64), InvalidScore(f64))}

{Contract:
  (In:  (bm25: f64(0..1))
        (vector: f64(0..1))
        (alpha: f64(0..1)))
  (Out: (score: f64(0..1)))
  (Inv: [score == (Add (Mul alpha vector) (Mul (Sub 1.0 alpha) bm25))])
}

{Logic:
  (Add (Mul alpha vector) (Mul (Sub 1.0 alpha) bm25))
}

{Checks:
  [(0.8, 0.6, 0.75) -> 0.65,
   (1.0, 1.0, 0.5)  -> 1.0,
   (0.0, 0.0, 0.5)  -> 0.0,
   (0.0, 1.0, 1.0)  -> 1.0,
   (1.0, 0.0, 0.0)  -> 1.0]
}
```

#### Comparison Notes

- The lambda-LLM version is more concise (15 lines vs 40 lines).
- The lambda-LLM version's invariant (`score == ...`) is machine-verifiable with Z3.
- The Structured Template version includes FILES_SCOPE, DEPENDS_ON, PRIORITY, and ESTIMATE — operational context that lambda-LLM does not address.
- The Structured Template version is readable by any developer without learning S-expression syntax.
- For *this specific task* (a pure mathematical function), lambda-LLM is the better fit.
- For the *broader project context* (where does this function live, what depends on it, who reviews it), the Structured Template provides necessary information that lambda-LLM omits.

**This is precisely why the hybrid approach (Section 7) is recommended: use the Structured Template for the operational envelope, embed the lambda-LLM node for the mathematical core.**
