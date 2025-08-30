# Coding Conventions

## Ruby Conventions

### File Headers

```ruby
# typed: strict
# frozen_string_literal: true
```

### String Quotes

- Always use double quotes for strings: `"example"`

### Hash Syntax

- Use shorthand notation: `{user:, name:}`
- NOT: `{user: user, name: name}`

### Private Methods

- Use `private def` syntax:

```ruby
private def process_value(value)
  value.upcase
end
```

### Conditionals

- No postfix if statements
- Always use full if/end blocks:

```ruby
# ✅ Good
if value.nil?
  return
end

# ❌ Bad
return if value.nil?
```

### Nil Handling

- Use `not_nil!` instead of `T.must`

### ActiveRecord

- Never use `includes`
- Use explicit `preload` or `eager_load`:

```ruby
Model.preload(:association)   # Separate queries (default)
Model.eager_load(:association) # JOIN (when filtering by association)
```

### Migrations

- Use ULID for primary keys:

```ruby
create_table :examples, id: false do |t|
  t.uuid :id, default: "generate_ulid()", null: false, primary_key: true
end
```

## RSpec Conventions

- No `context`, `let`, or `described_class`
- Define variables inside `it` blocks:

```ruby
it "xxxのとき、somethingすること" do
  user = FactoryBot.create(:user)
  # test implementation
end
```

## JavaScript/TypeScript Conventions

### HTTP Requests

- Never use `fetch` directly
- Always use `@rails/request.js`:

```typescript
import { post } from "@rails/request.js";

const response = await post("/api/endpoint", {
  body: data,
  responseKind: "json",
});
```

## Service Class Rules

### When to use Service classes

- ✅ Processes with database persistence
- ✅ Complex business logic across multiple models/records with persistence
- ✅ Transaction management needed

### When NOT to use Service classes

- ❌ Processes without database persistence (URL generation, data conversion)
- ❌ Single model/record operations (define as model or record methods)

### Transactions

- Always use `#with_transaction` method in Service classes
- Never use `ApplicationRecord.transaction` directly

## I18n

Translation files are organized by purpose:

- `forms.(ja,en).yml`: Form-related
- `messages.(ja,en).yml`: Messages and descriptions
- `meta.(ja,en).yml`: Metadata
- `nouns.(ja,en).yml`: Nouns and labels

## General Principles

- Avoid nested transactions
- Avoid record callbacks
- Prevent database access in Views/Components
- Descriptive naming conventions
- Comments in Japanese
- 100 characters per line maximum
- Follow security best practices
