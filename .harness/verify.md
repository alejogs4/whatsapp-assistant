# Verification Playbook (Tier 2 — BDD & Agentic Verification)

This playbook defines how agents verify completed tasks against their Gherkin `.feature` specs in this repository.

## 1. Tier 1 Deterministic Checks (`harness check`)

Always run deterministic checks first:

```bash
harness check
```

- Run a targeted subset with `harness check --only <name>` or machine-readable output with `harness check --json`.
- If any **required** Tier 1 check fails, fix it immediately before running Tier 2 verification.

## 2. Tier 2 BDD Acceptance Verification (`.feature` scenarios)

For each scenario in `.harness/work/<change-id>/specs/<task-id>.feature`:

1. **Prefer automated tests first**: If the scenario can be verified with a deterministic unit, integration, or executable BDD test, write and run that test.
2. **Verify every Given / When / Then**: Record concrete evidence per scenario:
   - **Scenario**: `<scenario title>`
   - **Status**: `PASS` | `FAIL`
   - **Evidence**: Command executed, test name, or observed output
3. **Spec gate**: `harness dag done <task-id>` validates that the referenced `.feature` file exists and contains valid `Feature:` and `Scenario:` declarations.

## 3. Sandbox / Computer-Use Playbook (`harness sandbox up/status/down`)

For flows requiring a running service, database, browser, or computer-use inspection:

```bash
harness sandbox up
harness sandbox status
# Execute Given / When / Then verification steps and capture evidence (HTTP responses, logs, screenshots)
harness sandbox down
```

- Configure `sandbox.up`, `sandbox.healthcheck`, `sandbox.healthcheck_timeout`, and `sandbox.down` in `.harness/checks.yaml`.
- Always tear down with `harness sandbox down` (automatic teardown also runs on healthcheck timeout or `SIGINT`/`SIGTERM`).

## 4. Tier 2 → Tier 1 Graduation Checklist

Whenever an agentic verification step stabilizes or repeats across tasks, propose promoting it into `.harness/checks.yaml`:

- [ ] Can the check be expressed as a single deterministic shell command with a reliable exit code?
- [ ] Does it run fast and cleanly without requiring LLM judgment?
- [ ] Is it repeatable across local dev and CI?
- [ ] Propose the YAML entry in your verification report for approval during `harness-distill`:

```yaml
  - name: <check-name>
    cmd: <deterministic-command>
    required: true
```
