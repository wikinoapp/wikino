# Suggested Commands for Wikino Development

## Environment Setup

```bash
docker compose up      # Start Docker environment
mise install          # Install dependencies
bin/setup            # Initial setup
bin/dev              # Start development environment
bin/rails server     # Start Rails server
```

## Development Commands

### Ruby/Rails Commands

```bash
bin/rails console     # Rails console
bin/rails generate    # Generate Rails files
bin/rails db:migrate  # Run database migrations
bin/rails db:rollback # Rollback migrations
bin/rails db:seed     # Seed database
```

### Testing

```bash
bin/rspec            # Run all tests
bin/rspec path/to/xxx_spec.rb  # Run specific test file
```

### Code Quality - Ruby

```bash
bin/standardrb       # Ruby linter and formatter
bin/erb_lint --lint-all  # ERB template linter
bin/srb tc          # Sorbet type checking
bin/rails sorbet:update  # Update Sorbet type definitions
bin/rails zeitwerk:check  # Check autoloading
```

### Code Quality - JavaScript/TypeScript

```bash
pnpm prettier . --write  # Format code with Prettier
pnpm eslint . --fix     # Lint and fix JavaScript/TypeScript
pnpm tsc               # TypeScript type checking
pnpm build            # Build JavaScript assets
pnpm build:css        # Build CSS with Tailwind
```

### Complete Verification

```bash
bin/check            # Run all verification checks
```

## Git Commands (Darwin/macOS)

```bash
git status          # Check status
git diff           # Show changes
git add .          # Stage changes
git commit -m "message"  # Commit changes
git push           # Push to remote
git pull           # Pull from remote
git log --oneline  # View commit history
```

## File System Commands (Darwin/macOS)

```bash
ls -la             # List files with details
cd path/to/dir     # Change directory
pwd               # Print working directory
mkdir dirname     # Create directory
touch filename    # Create file
rm filename       # Remove file
rm -rf dirname    # Remove directory
cp source dest    # Copy file
mv source dest    # Move/rename file
```

## Search Commands (Darwin/macOS)

```bash
find . -name "*.rb"  # Find files by pattern
grep -r "pattern" .  # Search in files
rg "pattern"        # Ripgrep (faster alternative)
```

## Process Management

```bash
ps aux | grep ruby  # Find Ruby processes
kill -9 PID        # Force kill process
lsof -i :3000      # Check what's using port 3000
```

## Docker Commands

```bash
docker compose up   # Start containers
docker compose down # Stop containers
docker compose ps   # List containers
docker compose logs # View logs
docker compose exec web bash  # Shell into container
```
