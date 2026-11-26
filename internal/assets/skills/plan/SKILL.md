---
name: rubio-plan
description: Plan and deliver large or risky changes in five short phases (explore, propose, spec and tasks, apply, verify). Use for new features, migrations, security, data or money changes, or anything that touches many files. Not for small edits.
---
<!-- rubio-ia:skill -->

# rubio-plan

Use this only when the change is large or risky. For anything smaller, follow the working agreement and skip it.

Name the change with a short slug (`add-billing-export`). Call it `<change>` below.

## Where artifacts go

- If the noah tools are available: `mem_save` with topic `plan/<change>/<phase>`. The same topic revises the note, so a phase can be redone without clutter.
- Otherwise: one file, `docs/plans/<change>.md`, with one section per phase.

Before starting, `mem_search` for `plan/<change>` in case work already exists, and resume from the last finished phase.

## Phases

1. **Explore** (read only). Read the code that matters. Write what exists, what constrains the change and what is unknown. No edits.
2. **Propose.** Goal, scope, what is out of scope, the chosen approach with one rejected alternative and why, and the risks. Include the design here; split it out only if the change is architectural. **Stop and wait for the user's approval.** This is the only mandatory pause.
3. **Spec and tasks.** Acceptance criteria as statements a test or a command can check. Then ordered tasks, each small enough to verify on its own.
4. **Apply.** Do the tasks in order and tick each one when its check passes. If reality differs from the plan, stop, update the plan, and tell the user what changed and why before going on.
5. **Verify.** Check every acceptance criterion with evidence (command output, test names). Report findings as CRITICAL (blocks), WARNING (real risk), SUGGESTION (optional). Do not mark done with an open CRITICAL.

## Independent review (high risk only)

For security, data, money or irreversible changes, hand the diff and the acceptance criteria to one fresh subagent and ask it to review without seeing your reasoning.

- Keep a findings ledger: id, severity, status (open, fixed, rejected with a reason).
- The re-review only checks the ledger items and the lines touched by the fixes. New observations about untouched lines are logged, they do not reopen the loop.
- At most one correction round. If it is still open, ask the user.

## Close

Save a short summary under `plan/<change>/summary`: decision, what changed, how it was verified, and anything left open.
