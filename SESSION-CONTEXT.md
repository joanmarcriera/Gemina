# SESSION CONTEXT / TOKEN MANAGEMENT RULE

At the start of each major task, and before accepting any new task or subtask, run /status and check the remaining context capacity for this Codex session.

If the remaining context capacity is low, do not continue with implementation work.

Use the following thresholds:

- If remaining context is above 25%: continue normally.
- If remaining context is between 15% and 25%: only continue if the next action is small and low-risk.
- If remaining context is below 15%: stop taking new tasks immediately.

When remaining context is below 15%, your only permitted actions are:

1. Stop all implementation or exploratory work.
2. Summarise the current state of the project.
3. List completed work.
4. List modified files.
5. List open issues, risks, and failing tests.
6. Provide the exact next prompt that should be used to continue in a fresh Codex session.

Do not start new coding, refactoring, debugging, research, or design work when below the threshold.

The continuation prompt must be self-contained and include:
- project goal
- current repository state
- important decisions already made
- files changed
- tests run and results
- remaining tasks in priority order
- known constraints
- any commands the next Codex session should run first


# HARD STOP RULE FOR LOW CONTEXT  (under 15%)

You must periodically check session context with /status.

Before every new task, subtask, file edit, test run, refactor, dependency change, or architectural decision, check whether the remaining context is sufficient.

If /status shows that remaining context capacity is below 15%, you must enter HANDOVER MODE.

In HANDOVER MODE you must not:
- edit files
- run long commands
- start debugging
- inspect unrelated files
- refactor
- install dependencies
- make design decisions
- accept any new user instruction except to produce a handover

In HANDOVER MODE you must only produce a concise but complete handover for a new Codex session.



# Always preserve enough context to hand over cleanly. Finishing with a good handover is more important than attempting one more task and losing the session context.
