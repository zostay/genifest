# Release Process

Guidelines for releasing new versions of Genifest.

!!! note "Work in Progress"
    This documentation page is being developed. Please check back soon for complete content.

## Release Workflow

Genifest follows semantic versioning and uses automated release processes.

### Version Numbering

We follow [Semantic Versioning](https://semver.org/):

- **MAJOR** - Incompatible API changes
- **MINOR** - New functionality (backward compatible)
- **PATCH** - Bug fixes (backward compatible)

Examples:

- `v1.0.0` - Major release
- `v1.1.0` - Minor release with new features
- `v1.0.1` - Patch release with bug fixes
- `v1.0.0-rc1` - Release candidate

### Release Types

**Release Candidates** (`-rc1`, `-rc2`):

- Pre-release versions for testing
- Feature-complete but may have bugs
- Published for community testing

**Stable Releases** (`v1.0.0`):

- Production-ready versions
- Thoroughly tested
- Recommended for production use

## Release Process

### 1. Preparation

1. **Update documentation**:
 
     - Ensure all new features are documented
     - Update CLI help text if needed
     - Review and update README.md

2. **Update version**:
   ```bash
   # Update version in internal/cmd/version.txt
   echo "v1.0.0" > internal/cmd/version.txt
   ```

3. **Update changelog**:
   ```bash
   # Add new section to Changes.md
   ```

### 2. Testing

1. **Run full test suite**:
   ```bash
   make check
   ```

2. **Test release build**:
   ```bash
   make release
   ```

3. **Manual testing**:
 
     - Test guestbook example
     - Verify CLI commands work
     - Test on different platforms

### 3. Release Creation

1. **Create release branch**:
   ```bash
   git checkout -b release/v1.0.0
   git add .
   git commit -m "chore: prepare v1.0.0 release"
   git push origin release/v1.0.0
   ```

2. **Create pull request**:
 
     - Review all changes
     - Ensure CI passes
     - Get approval from maintainers

3. **Merge and tag**:
   ```bash
   git checkout master
   git merge release/v1.0.0
   git tag v1.0.0
   git push origin master --tags
   ```

### 4. Automated Release

GitHub Actions automatically:

- Builds binaries for all platforms
- Generates checksums
- Creates GitHub release
- Uploads release artifacts

### 5. Post-Release

1. **Update documentation site**:

      - Documentation deploys automatically
      - Verify genifest.qubling.com is updated

2. **Announce release**:
 
      - GitHub release notes
      - Update package managers (if applicable)

## Release Checklist

### Pre-Release

- [ ] All tests pass
- [ ] Documentation is updated
- [ ] Version number is updated
- [ ] Changelog is updated
- [ ] Breaking changes are documented
- [ ] Dependencies are updated if needed

### Release

- [ ] Release branch created
- [ ] Pull request reviewed and approved
- [ ] Tag created and pushed
- [ ] GitHub release created
- [ ] Binaries are built and uploaded

### Post-Release

- [ ] Documentation site updated
- [ ] Release announced
- [ ] Issues closed if fixed
- [ ] Next milestone planned

## Branch Strategy

**Master Branch**:

- Always deployable
- All releases tagged from master
- Protected with required reviews

**Feature Branches**:

- Short-lived for specific features
- Merged via pull requests
- Deleted after merge

**Release Branches**:

- Created for release preparation
- Allows final fixes without blocking development
- Merged to master and deleted

## Hotfix Process

For critical bugs in released versions:

1. **Create hotfix branch from tag**:
   ```bash
   git checkout v1.0.0
   git checkout -b hotfix/v1.0.1
   ```

2. **Fix the issue**:
   ```bash
   # Make minimal changes to fix the bug
   git commit -m "fix: critical bug description"
   ```

3. **Release process**:
   ```bash
   # Update version to v1.0.1
   # Follow normal release process
   ```

4. **Merge back**:
   ```bash
   # Merge hotfix to master
   # Cherry-pick to development branches if needed
   ```

## Release Automation

### GitHub Actions

The release process is automated with GitHub Actions:

```yaml
# .github/workflows/release.yml
name: Release
on:
  push:
    tags:
      - 'v*'
      
jobs:
  release:
    runs-on: ubuntu-latest
    steps:
      - name: Build release binaries
        run: make release
      - name: Create GitHub release
        uses: actions/create-release@v1
```

### Build Matrix

Release builds target multiple platforms:

- Linux (amd64, arm64)
- macOS (amd64, arm64)
- Windows (amd64)

## Version Management

### Version File

Version is stored in `internal/cmd/version.txt`:
```
v1.0.0
```

### Build Integration

Version is embedded during build:
```go
// Embedded at build time
var Version = "dev"
var Commit = "unknown"
var BuildTime = "unknown"
```

### Makefile Integration

```makefile
VERSION := $(shell cat $(VERSION_FILE))
LDFLAGS := -ldflags "-X $(PACKAGE)/internal/cmd.Version=$(VERSION)"
```

## See Also

- [Contributing](contributing.md) - Development workflow
- [Testing](testing.md) - Testing requirements
- [Changelog](../changelog.md) - Release history