# CDN Distribution WAF Rule Lists — Fix Review Guide

**Issue:** [stackitcloud/terraform-provider-stackit#1630](https://github.com/stackitcloud/terraform-provider-stackit/issues/1630)
**Scope:** `stackit_cdn_distribution` resource, `config.waf.*` rule-list attributes
**Primary files touched:**

- `stackit/internal/services/cdn/distribution/resource.go`
- `stackit/internal/services/cdn/distribution/resource_test.go`
- `stackit/internal/utils/planmodifiers/setplanmodifier/empty_on_removal.go` (new)
- `stackit/internal/utils/planmodifiers/setplanmodifier/empty_on_removal_test.go` (new)
- `docs/resources/cdn_distribution.md`, `docs/data-sources/cdn_distribution.md` (regenerated)

---

## 1. The problem in a nutshell

The nine WAF rule-list attributes (`enabled/disabled/log_only` × `rule/rule_group/rule_collection` `_ids`) are declared `Optional: true, Computed: true`. That schema combination is the root of every symptom in this issue:

| # | Symptom | Trigger |
| --- | --------- | --------- |
| A | `terraform plan` reports "No changes" after a rule list is **removed** from the config | The framework carries the prior state value into the plan for unconfigured `Optional+Computed` attributes, so plan == state. |
| B | `Provider produced inconsistent result after apply`: `disabled_rule_ids: was cty.SetValEmpty(...), but now null` | The user works around (A) by setting `[]` explicitly; the apply clears the rules, but the server omits some (and only some!) of the fields if they are empty lists  and the provider maps the absent field back to `null`, contradicting the planned `[]`. |
| C | Error for `updated_at` on the removal-triggered update | The server bumps the timestamp on every PATCH, but the plan had carried over the old, known timestamp. |
| D | Error for `enabled_rule_collection_ids: was null, but now [...]` | When the WAF is enabled, the server injects default rule collections. The plan held `null` for the unconfigured list while the apply returned the defaults. |

The framework detail connecting (A), (C) and (D): the pass that marks null-config computed attributes as `(known after apply)` — `MarkComputedNilsAsUnknown` in `internal/fwserver/server_planresourcechange.go` (framework v1.19.0) — only runs **when the proposed new state already differs from prior state** (line 200 gate). When a removal is the *only* change, the proposed state equals prior state at that point, the pass is skipped, and every computed attribute keeps its prior state value as a *known* plan value. Plan modifiers run afterwards (line 293), so the diff created by our plan modifier comes too late to trigger the framework's own unknown-marking, which is why the `ModifyPlan` is needed.

### Verified API behavior (drove the design)

- `PATCH` with `"disabledRuleIds": []` clears the list; subsequent GET returns `[]`. This may not be true for other fields like `enabledRuleCollectionIds`, for which the server omits the field if configured as `[]`.
- Enabling the WAF (`mode = "ENABLED"`/`LOG_ONLY`) populates the `enabled_*` collections with server defaults **once**, when no user value exists. An explicit `[]` set by the user sticks and is not refilled.
- The `waf` config object is sparse when `mode = "DISABLED"`: rule-list fields are omitted from GET responses.
- The portal showing rules as "active" in the original report was a red herring: the WAF mode was `DISABLED`, so the portal showed its enable-feature page. Not a bug.

---

## 2. Design decisions

1. **Keep `Optional + Computed`.** The attributes genuinely have server-side defaults, so `Computed` is semantically correct.
2. **`enabled_*` vs. `disabled_*`/`log_only_*` behave differently on removal.**
   - `enabled_*` lists are **server-defaulted** on WAF enablement. Removal must mean "stop managing, accept the server value" — explicit `[]` still clears them. We have no control over when the server decides to inject it's own values.
   - `disabled_*` and `log_only_*` lists are never defaulted. Removal plans an empty set, i.e. actively clears the rules. This is the fix for symptom (A).
3. **The empty-vs-null ambiguity is resolved in favor of the plan.** Because the sparse-WAF GET response conflates "never configured" with "just cleared" (both absent), the provider preserves the value the plan/prior model carries when the API omits a field.
4. **Server-managed values are planned as unknown at the resource level**, not via per-attribute modifiers. The only reliable point to know "an update is happening" is after all attribute plan modifiers ran — a resource-level `ModifyPlan`. (An earlier draft per-attribute `updated_at` modifier was reverted for causing perpetual diffs.)

---

## 3. Implementation walkthrough

### 3.1 `setplanmodifier.EmptyOnRemoval()` — new plan modifier (symptom A)

`stackit/internal/utils/planmodifiers/setplanmodifier/empty_on_removal.go`

Plans an empty set when the attribute is removed from the configuration. Guard conditions:

- `req.State.Raw.IsNull()` → resource creation: nothing to clear, API defaults must apply.
- `!req.ConfigValue.IsNull()` → attribute still configured: leave the user's value alone.
- `req.ConfigValue.IsUnknown()` → interpolation safety.
- `req.StateValue.IsNull() || req.StateValue.IsUnknown()` → no prior value to clear (also protects never-configured server-defaulted lists).
- Plan already an empty set → no-op (idempotence, avoids perpetual diffs).

Critically, it fires both when the plan value is unknown (framework marked it) **and** when the plan value is a known carry-over of prior state — the latter being the exact "No changes" path of symptom (A).

Applied in `resource.go` schema to the six non-server-defaulted lists only: `disabled_rule_ids`, `log_only_rule_ids`, `disabled_rule_group_ids`, `log_only_rule_group_ids`, `disabled_rule_collection_ids`, `log_only_rule_collection_ids`. The three `enabled_*` attributes have **no** plan modifier (see decision 2).

### 3.2 `wafRuleListToSet()` — apply-time consistency (symptom B)

`resource.go`, helper used by `mapFields`.

```
apiList != nil → use the API value (populated or explicit [])
apiList == nil → keep the value already in the model
```

`mapFields` decodes the incoming model's `waf` object (`priorWaf`) and uses it as the fallback for all nine lists. Because the model is plan-derived in Create/Update, an explicitly planned `[]` survives as `[]` instead of becoming `null` — eliminating the inconsistency of symptom (B). In Read, the model is state-derived, so refresh preserves server values rather than nulling them.

### 3.3 `ModifyPlan` — resource-level unknown marking (symptoms C and D)

`resource.go`, `func (r *distributionResource) ModifyPlan` (registered via `resource.ResourceWithModifyPlan`).

Runs after all attribute plan modifiers, when "is this a real update?" is finally decidable:

1. **Skip on create** (`req.State.Raw.IsNull()`) — computed attributes are unknown there anyway.
2. **Skip on destroy** (`req.Plan.Raw.IsNull()`).
3. **Skip on no-op** (`planModel.UpdatedAt.Equal(stateModel.UpdatedAt) && planModel.Config.Equal(stateModel.Config)`) — this is the anti-perpetual-diff guard; note the comparison happens after `EmptyOnRemoval`, so a removal-only change *is* detected as an update.
4. On a real update:
   - `planModel.UpdatedAt = types.StringUnknown()` — the timestamp always changes on PATCH, so the plan must not assert the stale value (symptom C).
   - `markWafRuleListsUnknownIfUnconfigured(&planWaf, &configWaf)` — each of the nine lists that is null in **both** config and plan is set to unknown, so server-injected defaults (symptom D) are accepted on apply. Lists with a configured value or an `EmptyOnRemoval`-produced `[]` are untouched.

This replaces the framework's skipped `MarkComputedNilsAsUnknown` pass for exactly the attributes that need it, and composes with the framework's own pass when it does run (no-ops there).

### 3.4 Documentation

Attribute descriptions updated:

- `disabled_*`/`log_only_*`: "Set to an empty set **or remove the attribute** to clear previously set rules."
- `enabled_*`: "Set to an empty set to clear. When omitted, the server-managed set is left untouched (the API may populate defaults when the WAF is enabled)."

The old "retains the last known state" wording was removed.

---

## 4. Tests

- `empty_on_removal_test.go` — 9 cases: removal→empty (both unknown-plan and carried-over-known-plan paths), already-empty idempotence, configured values untouched, create/null-state guards.
- `TestMapFields` (`resource_test.go`) — two regression cases for symptom (B): API omits fields while the model holds explicit empty sets (state must stay empty), and API returns explicit empty lists (state must be empty regardless of prior populated model).
- `TestModifyPlan` (`resource_test.go`) — three scenarios: (1) update via WAF removal asserts `updated_at` and unconfigured `enabled_rule_collection_ids` become unknown while `disabled_rule_ids` stays `[]`; (2) no-op plan asserts nothing is touched (the anti-perpetual-diff guard); (3) create is skipped.

Each guard was validated by temporarily deleting it and confirming the corresponding test fails.

---

## 5. What to focus on when reviewing

1. **The no-op guard in `ModifyPlan`** (`planModel.Config.Equal(stateModel.Config)`) is load-bearing. Removing it reintroduces perpetual diffs on `updated_at` and all unconfigured WAF lists.
2. **The asymmetry between `enabled_*` and the other six lists** is intentional and tied to verified API behavior (defaults injected once, explicit `[]` sticks). "Unifiying" them has consequences. This should be discussed in the team, as users may expect each field to behave exactly the same.
3. **`wafRuleListToSet` trades drift detection for consistency.** If someone modifies WAF lists out of band and the API then omits a field (disabled-WAF sparse response), the provider keeps the last known value instead of surfacing drift. Explicitly taken trade-off during the implementation.
4. **Not covered by this change:** the three `allowed_*` sets keep `SizeAtLeast(1)` and unchanged semantics (they can never be cleared); other always-changing computed attributes (`status`, `errors`, `domains`) were left alone because the wait handler normally returns them in their final state — they might share the theoretical symptom-(C) exposure but were not explicitly tested and did not manifest it during the waf tests.
