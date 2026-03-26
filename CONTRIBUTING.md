# Contributing to customclaw

## Git workflow

This project uses **GitHub Flow**. It is simple: `main` is always deployable, and all work happens on short-lived feature branches.

```
main ──────────────────────────────────────────────────────▶
       \                    /      \                /
        feature/add-discord         fix/webhook-parse
```

### Step by step

1. **Create a branch from `main`**

   ```bash
   git checkout main
   git pull origin main
   git checkout -b feature/your-feature-name
   ```

   Branch naming:
   - `feature/` — new functionality
   - `fix/` — bug fix
   - `chore/` — tooling, deps, CI, docs
   - `refactor/` — internal restructure, no behavior change

2. **Make your changes**

   Keep commits small and focused. Write a clear commit message:
   - `feat: add Discord notification tool`
   - `fix: parse Jira webhook secret correctly`
   - `chore: update go.mod dependencies`
   - `docs: add CLI usage examples to README`

3. **Push and open a pull request to `main`**

   ```bash
   git push origin feature/your-feature-name
   ```

   Then open a PR on GitHub. The PR description should explain:
   - What the change does
   - Why it is needed
   - How to test it

4. **Get a review**

   At least one approval is required before merging.

5. **Merge and delete the branch**

   Use **squash merge** to keep `main` history clean. Delete the branch after merge.

### Rules

- Never commit directly to `main`
- Never force-push to `main`
- Keep feature branches short-lived (days, not weeks)
- `main` must always build and pass tests

## Development setup

```bash
git clone https://github.com/your-org/customclaw.git
cd customclaw
go mod tidy
cp config.example.json config.json
cp actions.example.json actions.json
# fill in your API keys in config.json
go build -o customclaw ./cmd
```

## Running tests

```bash
go test ./...
```

## Code style

- Follow standard Go conventions (`gofmt`, `golint`)
- Keep functions small and single-purpose
- Add a new tool by implementing the `Tool` interface in `internal/tools/`
- Add a new LLM provider by implementing the `Provider` interface in `internal/llm/`
