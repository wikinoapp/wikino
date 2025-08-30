# Task Completion Checklist

## Before Marking a Task as Complete

### After Writing/Modifying Ruby Code
1. Run linter: `bin/standardrb`
2. Run ERB linter: `bin/erb_lint --lint-all`
3. Run type checking: `bin/srb tc`
4. Update Sorbet types if needed: `bin/rails sorbet:update`
5. Check autoloading: `bin/rails zeitwerk:check`
6. Run related tests: `bin/rspec path/to/xxx_spec.rb`

### After Writing/Modifying JavaScript/TypeScript
1. Format code: `pnpm prettier . --write`
2. Run linter: `pnpm eslint . --fix`
3. Run type checking: `pnpm tsc`

### After Creating Tests
1. Run the test: `bin/rspec path/to/new_spec.rb`
2. Ensure test passes before completion

### After Editing Documentation
1. Format with Prettier: `pnpm prettier . --write`

### Complete Verification
- For comprehensive check: `bin/check`

## DO NOT Mark as Complete If:
- Tests are failing (unless intentionally creating failing tests for unimplemented features)
- Compilation errors exist
- Unresolved errors from previous attempts remain

## Retry Policy
- Automatically retry up to 5 times when errors occur
- Only report to user after 5 consecutive failures
- Do not report intermediate progress

## Important Notes
- Always verify changes work as expected
- Check for obvious runtime errors
- Ensure code follows project conventions
- Comments should be in Japanese
- Use double quotes for strings in Ruby
- Use `@rails/request.js` for HTTP requests in JavaScript
- Never use `includes` in ActiveRecord, use `preload` or `eager_load`
- Use `private def` for private methods
- Use `not_nil!` instead of `T.must`