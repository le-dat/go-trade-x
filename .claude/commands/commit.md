# /commit — Create Standard Git Commit

Create a git commit for current changes.

## Steps to Execute:

1. **Check changes:** Run `git status` and `git diff --staged` to understand what will be committed

2. **Write commit message** following Conventional Commits:
   ```
   <type>(<scope>): <brief description>

   [body — detailed description if needed]

   [footer — breaking changes, closes issues]
   ```

   Valid types:
   - `feat` — new feature
   - `fix` — bug fix
   - `docs` — documentation update
   - `style` — formatting, no logic change
   - `refactor` — code refactoring
   - `test` — add/modify tests
   - `chore` — build, dependencies

3. **Stage and commit:**
   ```bash
   git add -A
   git commit -m "<message>"
   ```

4. **DO NOT push** directly to main — create a PR instead

If you want to create a PR immediately after commit, use `/pr`.
