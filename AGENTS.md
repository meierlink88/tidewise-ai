# Project agent instructions

Before making or proposing any new change in this repository, invoke
`$eino-reference-first`. This applies to source code, tests, configuration,
documentation, prompts, skills, scripts, dependencies, generated artifacts,
and project structure. Complete the task-relevant audit of all three upstream
repositories even when the conclusion is that Eino is not needed.

Do not edit files until the skill's reference verification and three-repository
audit are complete. Read-only inspection and diagnostics may be performed first
when needed to determine the task-relevant search scope.

Treat `.reference/cloudwego/` as read-only upstream learning material. Never
edit, stage, commit, or publish files from that directory.
