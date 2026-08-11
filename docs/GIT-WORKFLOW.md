# Git Workflow & Branching Strategy

**Last Updated**: August 11, 2026

---

## Branch Structure

```
main (production-ready)
  └── dev (development)
       ├── feature/authentication
       ├── feature/fleet-management
       └── feature/trip-system
```

---

## Branches Explained

### `main` Branch
- **Purpose**: Production-ready code only
- **Protected**: Requires pull request + review
- **Deployment**: Automatically deployed to production (later)
- **Status**: Must always be stable and working

**Current State**: Phase 0 complete ✅

### `dev` Branch
- **Purpose**: Integration branch for all features
- **Testing**: Pre-production testing happens here
- **Daily Work**: Most development happens here
- **Merge From**: Feature branches
- **Merge To**: `main` (via PR when stable)

### `feature/*` Branches
- **Purpose**: Individual features or user stories
- **Naming**: `feature/short-description`
- **Lifetime**: Created from `dev`, merged back to `dev`
- **Examples**:
  - `feature/user-authentication`
  - `feature/jwt-middleware`
  - `feature/driver-crud`

### `bugfix/*` Branches
- **Purpose**: Non-urgent bug fixes
- **Naming**: `bugfix/short-description`
- **Created From**: `dev`
- **Merged To**: `dev`

### `hotfix/*` Branches
- **Purpose**: Critical production bugs
- **Naming**: `hotfix/short-description`
- **Created From**: `main`
- **Merged To**: Both `main` AND `dev`

---

## Development Workflow

### Starting New Feature (Phase 1+)

```bash
# 1. Switch to dev branch
git checkout dev

# 2. Pull latest changes
git pull origin dev

# 3. Create feature branch
git checkout -b feature/user-authentication

# 4. Work on feature, commit often
git add .
git commit -m "feat: add user registration endpoint"

# 5. Push to remote
git push -u origin feature/user-authentication

# 6. Create Pull Request: feature/user-authentication → dev
# 7. After review and tests pass, merge to dev
# 8. Delete feature branch
```

### Releasing to Production

```bash
# 1. On dev branch, ensure everything is tested
git checkout dev

# 2. Create Pull Request: dev → main
# 3. Review changes carefully
# 4. After approval, merge to main
# 5. Tag the release
git checkout main
git tag -a v0.2.0 -m "Phase 1: Authentication complete"
git push origin v0.2.0
```

---

## Initial Setup (Do This Now)

### Step 1: Push Current Work to Main

```bash
# You're on main with 9 unpushed commits
git push origin main
```

### Step 2: Create Dev Branch

```bash
# Create dev branch from current main
git checkout -b dev

# Push dev branch to remote
git push -u origin dev
```

### Step 3: Set Dev as Default Branch (Optional)

On GitHub/GitLab:
- Go to Settings → Branches
- Change default branch to `dev`
- This makes `dev` the landing page for PRs

---

## Common Commands

### Switch Branches
```bash
# Switch to dev
git checkout dev

# Switch to main
git checkout main

# Create and switch to feature branch
git checkout -b feature/my-feature
```

### Update Your Branch
```bash
# Pull latest changes
git pull origin dev

# If you have local changes, stash them first
git stash
git pull origin dev
git stash pop
```

### Sync Feature Branch with Dev
```bash
# You're on feature/my-feature
git checkout dev
git pull origin dev
git checkout feature/my-feature
git merge dev  # or: git rebase dev
```

### View Branches
```bash
# Local branches
git branch

# All branches (including remote)
git branch -a

# See which branch you're on
git status
```

---

## Commit Message Convention

Follow conventional commits format:

```
<type>(<scope>): <subject>

<body>

<footer>
```

### Types
- `feat:` New feature
- `fix:` Bug fix
- `docs:` Documentation changes
- `style:` Code style (formatting, no logic change)
- `refactor:` Code refactoring
- `test:` Adding tests
- `chore:` Maintenance tasks

### Examples
```bash
git commit -m "feat(auth): add JWT middleware"
git commit -m "fix(database): resolve connection pooling issue"
git commit -m "docs: update API documentation"
git commit -m "refactor(cache): simplify Redis client"
```

---

## Branch Protection Rules (Set These on GitHub/GitLab)

### For `main` Branch
- ✅ Require pull request before merging
- ✅ Require at least 1 approval
- ✅ Require status checks to pass
- ✅ Require branches to be up to date
- ✅ Require linear history (optional)
- ✅ Prevent direct pushes

### For `dev` Branch
- ✅ Require pull request before merging (recommended)
- ⚠️ Optionally allow direct pushes (for small team)

---

## Example: Phase 1 Development

### Week 1: User Authentication

```bash
# Start from dev
git checkout dev
git pull origin dev

# Create feature branch
git checkout -b feature/user-model

# Work and commit
git add models/user.go
git commit -m "feat(models): add User model with bcrypt"

git add repository/user_repository.go
git commit -m "feat(repository): add user repository with CRUD"

# Push and create PR
git push -u origin feature/user-model
# Create PR: feature/user-model → dev
```

### Week 2: JWT Middleware

```bash
# New feature branch
git checkout dev
git pull origin dev
git checkout -b feature/jwt-middleware

# Work and commit
git add middleware/auth.go
git commit -m "feat(middleware): add JWT authentication middleware"

git add middleware/auth_test.go
git commit -m "test(middleware): add JWT middleware tests"

# Push and create PR
git push -u origin feature/jwt-middleware
# Create PR: feature/jwt-middleware → dev
```

### End of Phase 1: Merge to Main

```bash
# All features merged to dev and tested
git checkout dev
git pull origin dev

# Create release PR: dev → main
# After approval and merge, tag it
git checkout main
git pull origin main
git tag -a v0.2.0 -m "Release v0.2.0: Authentication & User Management"
git push origin v0.2.0
```

---

## Handling Merge Conflicts

### During Feature → Dev Merge

```bash
# On feature branch
git checkout dev
git pull origin dev
git checkout feature/my-feature
git merge dev

# If conflicts occur
# 1. Open conflicted files
# 2. Resolve conflicts (look for <<<<<<, ======, >>>>>>)
# 3. Remove conflict markers
# 4. Test the code
# 5. Commit the resolution
git add .
git commit -m "merge: resolve conflicts with dev"
git push
```

---

## Pull Request Template

Create `.github/pull_request_template.md`:

```markdown
## Description
Brief description of changes

## Type of Change
- [ ] New feature
- [ ] Bug fix
- [ ] Breaking change
- [ ] Documentation update

## Testing
- [ ] Health check passes
- [ ] All tests pass
- [ ] Manual testing completed

## Checklist
- [ ] Code follows project style
- [ ] Added/updated tests
- [ ] Added/updated documentation
- [ ] No breaking changes (or documented)
- [ ] Reviewed own code
```

---

## Rollback Strategy

### Rollback Feature (Before Merge)
```bash
# Delete feature branch
git branch -D feature/bad-feature
git push origin --delete feature/bad-feature
```

### Rollback Commit (After Merge to Dev)
```bash
git checkout dev
git revert <commit-hash>
git push origin dev
```

### Emergency Rollback (Production)
```bash
# Revert to previous tag
git checkout main
git reset --hard v0.1.0
git push origin main --force  # Use with extreme caution!

# Better: Create hotfix
git checkout -b hotfix/revert-bad-change
git revert <commit-hash>
git push origin hotfix/revert-bad-change
# Create PR: hotfix → main
```

---

## Visual Git Log

```bash
# Beautiful commit history
git log --oneline --graph --decorate --all

# Last 10 commits
git log --oneline -10

# Commits by author
git log --author="Your Name"

# Commits in date range
git log --since="2026-08-01" --until="2026-08-11"
```

---

## Best Practices

### ✅ DO
- Commit often with clear messages
- Keep feature branches small (1-3 days of work)
- Pull from `dev` before creating feature branch
- Test locally before pushing
- Write descriptive PR descriptions
- Delete branches after merge
- Tag releases on `main`

### ❌ DON'T
- Don't commit directly to `main`
- Don't commit `.env` or secrets
- Don't force push to shared branches
- Don't create long-lived feature branches
- Don't merge without testing
- Don't commit broken code

---

## Team Workflow

### For Solo Developer (You)
```bash
# Simple workflow
dev → feature → dev → main
```

### For Small Team (2-4 people)
```bash
# Everyone creates feature branches from dev
dev → feature/person1 → dev
dev → feature/person2 → dev

# PR reviews before merge
```

### For Larger Team (5+ people)
```bash
# Add staging branch
main → staging → dev → feature
```

---

## Quick Reference

| Action | Command |
|--------|---------|
| Create feature branch | `git checkout -b feature/name` |
| Switch to dev | `git checkout dev` |
| Update current branch | `git pull origin $(git branch --show-current)` |
| Merge dev into feature | `git merge dev` |
| Push new branch | `git push -u origin branch-name` |
| Delete local branch | `git branch -d branch-name` |
| Delete remote branch | `git push origin --delete branch-name` |
| View all branches | `git branch -a` |
| View current branch | `git branch --show-current` |

---

## Related Documentation

- [Contributing Guidelines](../CONTRIBUTING.md)
- [Developer Guide](DEVELOPER-GUIDE.md)
- [Quick Reference](QUICK-REFERENCE.md)

---

**Next Steps**: 
1. Push current work to `main`
2. Create `dev` branch
3. Set up branch protection rules on GitHub/GitLab
4. Start Phase 1 development on feature branches
