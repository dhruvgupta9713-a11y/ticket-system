# Contributing to Ticket System

Thank you for your interest in contributing to the Ticket System project!

## Getting Started

1. Fork the repository
2. Clone your fork locally
3. Create a new branch for your feature or bug fix
4. Make your changes
5. Run the tests to ensure nothing is broken
6. Submit a pull request

## Development Setup

```bash
# Install Go dependencies
go mod download

# Run the server locally
go run cmd/server/main.go

# Run with Docker
docker build -t ticket-system .
docker run -p 8080:8080 ticket-system
```

## Code Style

- Follow standard Go conventions and formatting (`gofmt`)
- Write meaningful commit messages
- Add comments for complex logic
- Ensure all exported functions have documentation

## Reporting Issues

- Use GitHub Issues to report bugs
- Include steps to reproduce the issue
- Include expected vs actual behavior
- Include Go version and OS information

## License

By contributing, you agree that your contributions will be licensed under the same license as the project.
