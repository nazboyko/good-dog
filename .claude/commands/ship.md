Run the full ship sequence for the current change:
1. review the diff with fresh eyes as an independent pass: remove anything unused, simplify anything overcomplicated, check naming, confirm no gameplay logic sits in React components
2. if anything generated or player facing changed, get a narrative-guardian review and act on it
3. run affected tests, then the full suite
4. if prompts or narrative changed, run the grounding evals, zero new hallucinations
5. write one commit message in project style from CLAUDE.md and commit
6. push
Stop and report if any step fails. Never skip a step to save time, never batch several units into one ship.
