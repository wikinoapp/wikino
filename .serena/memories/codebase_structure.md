# Codebase Structure

## App Directory Architecture
Wikino follows a specific architectural pattern with clear responsibilities:

| Directory | Responsibility | Description |
|-----------|---------------|-------------|
| **controllers/** | HTTP request handling | 1 action per controller, implements `#call` method |
| **records/** | DB table operations | ActiveRecord::Base inheritance, 1 table per record |
| **models/** | Domain logic | PORO, no database access |
| **repositories/** | Data conversion | Converts between Record and Model |
| **services/** | Business logic | **Only for processes with data persistence** |
| **forms/** | Form processing | Validation and data conversion |
| **components/** | UI components | ViewComponent, reusable UI elements |
| **views/** | Views | Uses ViewComponent, DB direct access prohibited |
| **policies/** | Authorization rules | Permission management |
| **validators/** | Custom validation | ActiveModel validator extensions |
| **jobs/** | Async processing | Minimal logic, mainly calls Services |
| **mailers/** | Email sending | Action Mailer |

## Class Dependencies Rules
Strict dependency rules to maintain clean architecture:

| Class | Can depend on |
|-------|---------------|
| Component | Component, Form, Model |
| Controller | Form, Model, Record, Repository, Service, View |
| Form | Record, Validator |
| Job | Service |
| Mailer | Model, Record, Repository, View |
| Model | Model |
| Policy | Record |
| Record | Record |
| Repository | Model, Record |
| Service | Job, Mailer, Record |
| Validator | Record |
| View | Component, Form, Model |

## Naming Conventions
- Controller: `(ModelPlural)::(ActionName)Controller`
- Service: `(ModelPlural)::(Verb)Service`
- Form: `(ModelPlural)::(Noun)Form`
- Repository: `(Model)Repository`
- View: `(ModelPlural)::(ActionName)View`
- Component: `(UIComponentPlural)::(Noun)Component`