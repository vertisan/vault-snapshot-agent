# Semantic Release Setup

This repository now uses [semantic-release](https://semantic-release.gitbook.io/) to automate the release process based on conventional commit messages.

## How It Works

1. **Commit Convention**: Use [Conventional Commits](https://www.conventionalcommits.org/) format for your commit messages:
   - `feat:` - A new feature (triggers a minor version bump)
   - `fix:` - A bug fix (triggers a patch version bump)
   - `docs:` - Documentation changes (no version bump)
   - `chore:` - Maintenance tasks (no version bump)
   - `refactor:` - Code refactoring (no version bump)
   - `perf:` - Performance improvements (triggers a patch version bump)
   - `test:` - Adding or updating tests (no version bump)
   - `BREAKING CHANGE:` - Breaking changes (triggers a major version bump)

2. **Automated Release Process**:
   - When commits are pushed to `main` or `master` branch, the semantic-release workflow runs
   - It analyzes commit messages to determine the next version
   - If a release is needed:
     - Creates a new git tag
     - Generates/updates CHANGELOG.md
     - Creates a GitHub release with release notes
   - The tag creation triggers the release workflow
   - The release workflow builds binaries using GoReleaser and attaches them to the GitHub release

## Workflow Files

- `.github/workflows/semantic-release.yaml` - Runs semantic-release on pushes to main/master
- `.github/workflows/release.yaml` - Builds and releases binaries when tags are created
- `.github/workflows/lint.yaml` - Runs linting and tests on all branches
- `.releaserc.json` - Configuration for semantic-release

## Example Commits

```bash
# Feature (minor version bump)
git commit -m "feat: add support for GCS storage backend"

# Bug fix (patch version bump)
git commit -m "fix: resolve snapshot retention cleanup issue"

# Breaking change (major version bump)
git commit -m "feat: redesign configuration format

BREAKING CHANGE: configuration file format has changed from YAML to JSON"

# Documentation (no version bump)
git commit -m "docs: update README with new storage options"
```

## Running a Release

Simply push commits with conventional commit messages to the main/master branch:

```bash
git commit -m "feat: add new feature"
git push origin master
```

The semantic-release workflow will automatically:
1. Determine if a release is needed
2. Calculate the next version number
3. Create a tag and GitHub release
4. Trigger the binary build process

## Manual Release

You can manually trigger the semantic-release workflow from the GitHub Actions tab if needed.
