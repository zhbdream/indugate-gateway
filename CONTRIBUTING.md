# Contributing to InduGate

Thank you for your interest in contributing!

## Development Setup

```bash
make deps
mkdir -p data
make run

# Frontend (separate terminal)
cd web && npm install && npm run dev
```

## Code Style

- Go: follow standard `gofmt` formatting (`make fmt`)
- Commits: use [Conventional Commits](https://www.conventionalcommits.org/)
- Tests: run `make test` before submitting PRs

## Pull Request Process

1. Fork the repository and create a feature branch
2. Add tests for new functionality when applicable
3. Update documentation if you change APIs or behavior
4. Ensure `go test ./...` and `cd web && npm run build` pass
5. Open a PR with a clear description of changes

## Reporting Issues

Please include:
- InduGate version or commit hash
- Steps to reproduce
- Expected vs actual behavior
- Relevant logs
