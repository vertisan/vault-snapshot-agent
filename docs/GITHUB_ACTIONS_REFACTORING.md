# GitHub Actions Refactoring Summary

This document summarizes the changes made to refactor the GitHub Actions workflows for the vault-snapshot-agent repository.

## Changes Made

### 1. Replaced Personal Runners with Default GitHub-Hosted Runners

All workflow files have been updated to use `ubuntu-latest` instead of the custom `personal` runner:

- **`.github/workflows/lint.yaml`**: Updated 3 jobs (linter, tests, go-releaser) to use `ubuntu-latest`
- **`.github/workflows/release.yaml`**: Updated release job to use `ubuntu-latest`
- **`.github/workflows/semantic-release.yaml`**: New workflow uses `ubuntu-latest`

### 2. Improved Workflow Structure

The existing **lint.yaml** workflow already has a good structure with:
- ✅ **Linter job**: Runs `golangci-lint` to check code quality
- ✅ **Tests job**: Runs unit tests with `gotestsum` and generates test summary
- ✅ **GoReleaser validation job**: Validates `.goreleaser.yaml` configuration

This structure follows Go application best practices.

### 3. Added Semantic Release

Implemented automated release management using semantic-release:

#### New Files Created:
1. **`.releaserc.json`** - Semantic release configuration
   - Uses conventional commits preset
   - Generates CHANGELOG.md
   - Creates GitHub releases
   - Commits CHANGELOG.md back to repository

2. **`.github/workflows/semantic-release.yaml`** - Semantic release workflow
   - Triggers on pushes to `main` or `master` branches
   - Can be manually triggered via workflow_dispatch
   - Analyzes commits to determine version bumps
   - Creates git tags automatically
   - Generates release notes

3. **`docs/SEMANTIC_RELEASE.md`** - Documentation
   - Explains how semantic-release works
   - Provides examples of conventional commits
   - Describes the release process

#### How It Works:
1. Developers commit using conventional commit format (`feat:`, `fix:`, etc.)
2. When pushed to main/master, semantic-release workflow runs
3. Semantic-release analyzes commits and determines if a release is needed
4. If yes, it creates a tag and GitHub release
5. The tag triggers the existing `release.yaml` workflow
6. GoReleaser builds and publishes binaries

### 4. Validation

All changes have been validated:
- ✅ YAML syntax validated for all workflow files
- ✅ JSON syntax validated for `.releaserc.json`
- ✅ Workflows validated with `actionlint` (no errors)
- ✅ All unit tests pass

## Workflow Diagram

```
Developer commits with conventional commit message
          ↓
    Push to main/master
          ↓
[semantic-release.yaml runs]
          ↓
  Analyzes commits → Determines version
          ↓
    Creates git tag
          ↓
  [release.yaml triggers]
          ↓
    GoReleaser builds binaries
          ↓
    Binaries attached to GitHub release
```

## Benefits

1. **No Dependency on Personal Runners**: All workflows now run on GitHub-hosted runners
2. **Automated Release Process**: No manual version management needed
3. **Consistent Versioning**: Follows semantic versioning automatically
4. **Automated Changelog**: CHANGELOG.md is generated from commit messages
5. **Better CI/CD**: Clear separation between linting/testing and releasing

## Migration Notes

- The existing tag-based release workflow (`release.yaml`) remains unchanged
- It will work seamlessly with semantic-release created tags
- No breaking changes to the existing release process
- Teams should start using conventional commit messages for automatic releases
