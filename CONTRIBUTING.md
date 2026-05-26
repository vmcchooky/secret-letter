# Contributing to One-Time Link

Thank you for your interest in contributing to One-Time Link! This document provides guidelines for contributing to the project.

## Project Status

**Current Status:** Production-ready, Milestone 4 complete  
**Next Milestone:** Production Deployment (Milestone 5)

## How to Contribute

### Reporting Bugs

If you find a bug, please open an issue on GitHub with:
- Clear description of the bug
- Steps to reproduce
- Expected behavior
- Actual behavior
- Environment details (OS, Go version, Node version)
- Screenshots if applicable

### Suggesting Features

Feature suggestions are welcome! Please open an issue with:
- Clear description of the feature
- Use case and benefits
- Proposed implementation (if you have ideas)
- Any relevant examples or references

### Code Contributions

We welcome code contributions! Here's how to get started:

#### 1. Fork and Clone

```bash
# Fork the repository on GitHub
# Then clone your fork
git clone https://github.com/YOUR_USERNAME/one-time-link.git
cd one-time-link
```

#### 2. Set Up Development Environment

Follow the instructions in [DEVELOPMENT.md](DEVELOPMENT.md):

```bash
# Start Redis
docker compose -f deploy/local/docker-compose.yml up -d

# Start backend
go run ./backend/cmd/api

# Start frontend (in another terminal)
cd frontend/web-app
npm install
npm run dev
```

#### 3. Create a Branch

```bash
git checkout -b feature/your-feature-name
# or
git checkout -b fix/your-bug-fix
```

#### 4. Make Your Changes

- Follow the existing code style
- Write clear, descriptive commit messages
- Add tests for new features
- Update documentation as needed

#### 5. Test Your Changes

```bash
# Backend tests
go test ./...

# Integration tests
go test ./backend/test -v

# Load tests (if applicable)
./scripts/load-test.sh --concurrent 10 --requests 100
```

#### 6. Commit and Push

```bash
git add .
git commit -m "feat: add your feature description"
# or
git commit -m "fix: fix your bug description"

git push origin feature/your-feature-name
```

#### 7. Create Pull Request

- Go to GitHub and create a Pull Request
- Provide clear description of changes
- Reference any related issues
- Wait for review

## Code Style Guidelines

### Go (Backend)

- Follow standard Go conventions
- Use `gofmt` for formatting
- Write clear, descriptive function names
- Add comments for exported functions
- Keep functions small and focused
- Use meaningful variable names

### TypeScript/React (Frontend)

- Follow TypeScript best practices
- Use functional components with hooks
- Write clear, descriptive component names
- Add JSDoc comments for complex functions
- Keep components small and focused
- Use meaningful variable names

### General

- Write self-documenting code
- Add comments for complex logic
- Keep lines under 100 characters when possible
- Use consistent naming conventions
- Avoid premature optimization

## Commit Message Guidelines

We follow conventional commits format:

```
<type>(<scope>): <subject>

<body>

<footer>
```

**Types:**
- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation changes
- `style`: Code style changes (formatting, etc.)
- `refactor`: Code refactoring
- `test`: Adding or updating tests
- `chore`: Maintenance tasks

**Examples:**
```
feat(api): add rate limiting to create endpoint
fix(frontend): fix decryption error handling
docs(readme): update installation instructions
test(backend): add integration tests for consume endpoint
```

## Testing Requirements

All contributions should include appropriate tests:

### Backend
- Unit tests for new functions
- Integration tests for new endpoints
- Update existing tests if behavior changes

### Frontend
- Component tests (when test framework is added)
- Manual testing for UI changes
- Cross-browser testing for critical features

## Documentation Requirements

Update documentation when:
- Adding new features
- Changing existing behavior
- Modifying API contracts
- Updating deployment procedures

**Files to update:**
- README.md (if user-facing changes)
- API documentation (if API changes)
- Code comments (for complex logic)
- Milestone documentation (if applicable)

## Review Process

1. **Automated Checks**
   - All tests must pass
   - Code must compile without errors
   - No security vulnerabilities (govulncheck)

2. **Code Review**
   - Maintainers will review your code
   - Address any feedback or questions
   - Make requested changes if needed

3. **Approval and Merge**
   - Once approved, your PR will be merged
   - Your contribution will be credited

## Questions?

If you have questions about contributing:

- Check [DEVELOPMENT.md](DEVELOPMENT.md) for setup help
- Review existing issues and PRs
- Open a discussion on GitHub
- Contact us at contact@quorix.io.vn

## Code of Conduct

### Our Standards

- Be respectful and inclusive
- Welcome newcomers
- Accept constructive criticism
- Focus on what's best for the project
- Show empathy towards others

### Unacceptable Behavior

- Harassment or discrimination
- Trolling or insulting comments
- Personal or political attacks
- Publishing others' private information
- Other unprofessional conduct

## License

By contributing to One-Time Link, you agree that your contributions will be licensed under the MIT License.

## Recognition

Contributors will be recognized in:
- GitHub contributors list
- Release notes (for significant contributions)
- Project documentation (for major features)

## Thank You!

Thank you for contributing to One-Time Link! Your contributions help make this project better for everyone.

---

**Developed by:** Quorix Việt Nam

- **Website:** [quorix.io.vn](https://quorix.io.vn)
- **Email:** contact@quorix.io.vn
- **Facebook:** [facebook.com/quorixvietnam](https://facebook.com/quorixvietnam)
