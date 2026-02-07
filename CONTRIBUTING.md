# Contributing to Okapi

Thank you for your interest in contributing to Okapi! We welcome contributions from the community and are grateful for your support.

This document provides guidelines and instructions for contributing to the project.


## Code of Conduct

This project adheres to a code of conduct that all contributors are expected to follow. Please be respectful, inclusive, and considerate in all interactions.

**In short:**
- Be respectful and welcoming
- Be patient and helpful
- Focus on what's best for the community
- Show empathy towards others

---

## How Can I Contribute?

There are many ways to contribute to Okapi:

- **Report bugs** – Help us identify and fix issues
- **Suggest features** – Share ideas for improvements
- **Write documentation** – Improve guides, examples, and API docs
- **Submit code** – Fix bugs, add features, or improve performance
- **Review pull requests** – Provide feedback on proposed changes
- **Share examples** – Create tutorials or sample applications
- **Answer questions** – Help others in discussions and issues

---

## Getting Started

### Prerequisites

- **Go 1.21 or higher** – [Install Go](https://golang.org/doc/install)
- **Git** – [Install Git](https://git-scm.com/downloads)
- **A GitHub account** – [Sign up](https://github.com/join)

### Fork and Clone

1. **Fork the repository** on GitHub
2. **Clone your fork locally:**
````bash
git clone https://github.com/jkaninda/okapi.git
cd okapi
````

3. **Add the upstream remote:**
````bash
git remote add upstream https://github.com/jkaninda/okapi.git
````

4. **Install dependencies:**
````bash
go mod download
````

5. **Verify everything works:**
````bash
 go test
````

---

## Development Workflow

### 1. Create a Branch

Always create a new branch for your work:
````bash
git checkout -b feature/your-feature-name
# or
git checkout -b fix/your-bug-fix
````

**Branch naming conventions:**
- `feature/` – New features
- `fix/` – Bug fixes
- `docs/` – Documentation updates
- `refactor/` – Code refactoring
- `test/` – Test improvements
- `chore/` – Maintenance tasks

### 2. Make Your Changes

- Write clean, readable code
- Follow the coding standards (see below)
- Add tests for new functionality
- Update documentation as needed
- Keep commits focused and atomic

### 3. Test Your Changes
````bash
# Run all tests
go test ./...

# Run tests with coverage
go test -coverprofile=coverage.txt

# View coverage report
go tool cover -html=coverage.txt
````

### 4. Keep Your Branch Updated

Regularly sync with upstream to avoid conflicts:
````bash
git fetch upstream
git rebase upstream/main
````

## Community

### Getting Help

- **GitHub Discussions** – Ask questions, share ideas
- **GitHub Issues** – Report bugs, request features
- **LinkedIn** – Connect with the maintainer: [Jonas Kaninda](https://www.linkedin.com/in/jkaninda/)

### Staying Updated

- **Watch the repository** for notifications
- **Star the repository** to show support
- **Follow releases** for new versions

---

## Questions?

If you have questions about contributing, feel free to:

- Open a discussion on GitHub
- Comment on an existing issue
- Reach out to the maintainer

---

## License

By contributing to Okapi, you agree that your contributions will be licensed under the MIT License.

---

<div align="center">

**Thank you for contributing to Okapi! 🎉**

**Made with ❤️ by the community**

</div>