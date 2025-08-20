# Contributing

Welcome to the Genifest project! We appreciate your interest in contributing.

!!! note "Work in Progress"
    This documentation page is being developed. Please check back soon for complete content.

## Getting Started

### Prerequisites

- Go 1.24 or later
- Git
- Make (optional but recommended)

### Development Setup

1. **Fork and clone the repository**:
   ```bash
   git clone https://github.com/yourusername/genifest.git
   cd genifest
   ```

2. **Install dependencies**:
   ```bash
   make deps
   # or manually:
   go mod download
   ```

3. **Install development tools**:
   ```bash
   make tools
   ```

4. **Run tests**:
   ```bash
   make test
   ```

5. **Build the project**:
   ```bash
   make build
   ```

## Development Workflow

### Making Changes

1. **Create a feature branch**:
   ```bash
   git checkout -b feature/your-feature-name
   ```

2. **Make your changes**

3. **Run quality checks**:
   ```bash
   make check  # Runs fmt, vet, lint, and test
   ```

4. **Commit your changes**:
   ```bash
   git add .
   git commit -m "feat: add your feature description"
   ```

### Commit Message Format

We follow conventional commit format:

- `feat:` - New features
- `fix:` - Bug fixes
- `docs:` - Documentation updates
- `test:` - Test additions/updates
- `chore:` - Maintenance tasks

### Pull Request Process

1. **Push your branch**:
   ```bash
   git push origin feature/your-feature-name
   ```

2. **Create a pull request**

3. **Address review feedback**

4. **Ensure CI passes**

## Code Style

- Follow Go conventions and best practices
- Use `gofmt` for formatting
- Run `golangci-lint` before submitting
- Write tests for new functionality
- Add documentation for user-facing features

## Testing

### Running Tests

```bash
# All tests
make test

# Tests with coverage
make test-coverage

# Race detection
make test-race

# Short tests only
make test-short
```

### Writing Tests

- Write unit tests for new functions
- Add integration tests for complex features
- Use the guestbook example for end-to-end testing

## Documentation

### Building Documentation

```bash
# Install docs dependencies
make docs-install

# Serve locally
make docs-serve

# Build static site
make docs-build
```

### Documentation Guidelines

- Update documentation for user-facing changes
- Use clear, concise language
- Provide examples where helpful
- Test documentation locally before submitting

## Release Process

Releases are handled by maintainers. Contributors should:

- Update CHANGELOG.md for significant changes (use the heading `## WIP  TBD` to start a new section at the top of the file)
- Ensure version compatibility
- Update documentation as needed

## Getting Help

- **GitHub Issues**: Report bugs or request features
- **GitHub Discussions**: Ask questions or discuss ideas
- **Documentation**: Check existing documentation first

## Code of Conduct

We expect all contributors to follow our code of conduct:

- Be respectful and inclusive
- Focus on constructive feedback
- Help create a welcoming environment

## See Also

- [Architecture](architecture.md) - System architecture
- [Testing](testing.md) - Testing guidelines
- [Release Process](releases.md) - Release procedures