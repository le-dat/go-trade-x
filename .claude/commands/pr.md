# /pr — Create Pull Request to GitHub

Create a Pull Request from the current branch.

## Steps to Execute:

1. **Check branch:** Ensure you're not on `main`
   ```bash
   git branch --show-current
   ```

2. **Push branch to remote:**
   ```bash
   git push origin $(git branch --show-current)
   ```

3. **Create PR using GitHub CLI:**
   ```bash
   gh pr create \
     --title "<title>" \
     --body "<body>" \
     --base main
   ```

4. **PR body** should include:
   - **Summary:** Brief description of changes
   - **Changes:** List of specific changes
   - **Testing:** How to test the feature
   - **Screenshots:** (if UI changes)
   - **Closes:** Which issue to close (example: `Closes #42`)

5. Print the PR link after creation.
