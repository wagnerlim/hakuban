---
name: project-board
key: DEMO
columns:
    - To-Do
    - Doing
    - Review
    - Done
actions:
    Doing:
        guide: One card at a time here.
        loader: segments
        percent: true
        on_enter_cmd: hooks/dispatch.sh
        on_exit_cmd: hooks/handoff.sh
    Review:
        guide: Nothing leaves here without a PR.
        on_exit_cmd: hooks/require-pr.sh
    Done:
        percent: false
agent_tools: Read, Write, Bash
---

Recording fixture for the homepage hero. Scripts only: the instruction path needs a paid
Claude Code subscription, so it cannot run on a clean machine.
