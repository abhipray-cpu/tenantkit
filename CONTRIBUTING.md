# Contributing to TenantKit

First off, thank you for considering contributing to TenantKit! 🎉

This project is maintained by a single individual, and every contribution is greatly appreciated. Here's how you can help make TenantKit better.

## Table of Contents

- [Code of Conduct](#code-of-conduct)
- [Getting Started](#getting-started)
- [How Can I Contribute?](#how-can-i-contribute)
- [Development Setup](#development-setup)
- [Pull Request Process](#pull-request-process)
- [Style Guidelines](#style-guidelines)
- [Community](#community)

## Code of Conduct

This project follows a simple code of conduct: **Be respectful and constructive.** We're all here to learn and build something useful together.

- Be welcoming to newcomers
- Be patient when explaining concepts
- Focus on what's best for the community
- Show empathy towards other community members

## Getting Started

### Prerequisites

- Go 1.21 or higher
- Docker and Docker Compose (for testing)
- Git

### Fork and Clone

1. Fork the repository on GitHub
2. Clone your fork:
   ```bash
   git clone https://github.com/YOUR_USERNAME/tenantkit.git
   cd tenantkit
   ```
3. Add the upstream remote:
   ```bash
   git remote add upstream https://github.com/abhipray-cpu/tenantkit.git
   ```

## How Can I Contribute?

### 🐛 Reporting Bugs

Before creating a bug report, please check if the issue already exists. When creating a bug report, include:

- **Clear title** describing the issue
- **Steps to reproduce** the behavior
- **Expected behavior** vs **actual behavior**
- **Go version** and **OS**
- **Relevant code snippets** (minimal reproducible example)

Use this template:

```markdown
**Describe the bug**
A clear and concise description.

**To Reproduce**
1. Configure tenantkit with...
2. Execute query...
3. See error

**Expected behavior**
What you expected to happen.

**Environment:**
- OS: [e.g., macOS 14.0]
- Go version: [e.g., 1.21.5]
- TenantKit version: [e.g., v0.1.0]
- Database: [e.g., PostgreSQL 16]

**Additional context**
Any other relevant information.
```

### 💡 Suggesting Enhancements

Enhancement suggestions are welcome! Please include:

- **Use case**: Why is this feature needed?
- **Proposed solution**: How should it work?
- **Alternatives considered**: Other approaches you've thought of
- **Additional context**: Examples, mockups, references

### 📝 Improving Documentation

Documentation improvements are always welcome:

- Fix typos and grammatical errors
- Add missing examples
- Clarify confusing sections
- Add translations

### 🔧 Code Contributions

Here are some areas where contributions are especially welcome:

- **New adapters**: Database drivers, HTTP frameworks, cache backends
- **Performance improvements**: Benchmarks, optimizations
- **Test coverage**: Edge cases, integration tests
- **Bug fixes**: Check the issue tracker

## Development Setup

### 1. Install Dependencies

```bash
# Clone the repository
git clone https://github.com/abhipray-cpu/tenantkit.git
cd tenantkit

# Download dependencies
go mod download

# Start test infrastructure
docker-compose up -d postgres redis
```

### 2. Run Tests

```bash
# Unit tests
go test ./tenantkit/...

# Integration tests (requires Docker)
go test ./testing/integration/...

# All tests
make test

# With coverage
make coverage
```

### 3. Run Linter

```bash
# Install golangci-lint
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Run linter
golangci-lint run
```

### 4. Build

```bash
# Verify everything compiles
go build ./...
```

## Pull Request Process

### 1. Create a Branch

```bash
git checkout -b feature/your-feature-name
# or
git checkout -b fix/issue-description
```

### 2. Make Changes

- Follow the [style guidelines](#style-guidelines)
- Write/update tests for your changes
- Update documentation if needed
- Keep commits atomic and well-described

### 3. Commit Messages

Follow [Conventional Commits](https://www.conventionalcommits.org/):

```
<type>(<scope>): <description>

[optional body]

[optional footer]
```

Types:
- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation only
- `style`: Formatting, no code change
- `refactor`: Code restructuring
- `perf`: Performance improvement
- `test`: Adding tests
- `chore`: Maintenance tasks

Examples:
```
feat(cache): add Redis cluster support

fix(wrap): handle nil context gracefully

docs(readme): add installation instructions

perf(query): implement query caching
```

### 4. Push and Create PR

```bash
git push origin feature/your-feature-name
```

Then create a Pull Request on GitHub with:

- **Clear title** following commit message format
- **Description** of what changes and why
- **Link to related issues** (Fixes #123)
- **Screenshots** if applicable (for documentation)

### 5. Review Process

- A maintainer will review your PR
- Address any feedback
- Once approved, your PR will be merged

## Style Guidelines

### Go Code Style

Follow standard Go conventions:

```go
// Good: Clear function name, proper error handling
func (db *DB) Query(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
    if ctx == nil {
        return nil, ErrNilContext
    }
    
    tenantID, ok := GetTenant(ctx)
    if !ok {
        return nil, ErrMissingTenant
    }
    
    // ... implementation
}

// Good: Exported functions have doc comments
// Query executes a SELECT query with automatic tenant filtering.
// The tenant ID is extracted from context and injected into the WHERE clause.
func (db *DB) Query(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
```

### Testing

```go
// Good: Table-driven tests with clear names
func TestDB_Query(t *testing.T) {
    tests := []struct {
        name      string
        query     string
        tenantID  string
        want      string
        wantErr   bool
    }{
        {
            name:     "simple select with tenant",
            query:    "SELECT * FROM users",
            tenantID: "tenant-1",
            want:     "SELECT * FROM users WHERE tenant_id = $1",
        },
        // ... more cases
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // ... test implementation
        })
    }
}
```

### Documentation

- Use clear, concise language
- Include code examples
- Keep examples runnable (use `go run` or Go Playground links)

## Community

### Getting Help

- **GitHub Discussions**: Ask questions, share ideas
- **GitHub Issues**: Report bugs, request features
- **Email**: dumkaabhipray@gmail.com (for sensitive matters)

### Recognition

Contributors are recognized in:
- `CONTRIBUTORS.md` file
- Release notes
- GitHub's contributor graph

## Thank You! 🙏

Your contributions make TenantKit better for everyone. Whether it's a bug report, documentation fix, or new feature - every contribution matters!

---

*This contributing guide is adapted from various open source projects and tailored for TenantKit's needs.*
