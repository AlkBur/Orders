# RFC-016: Field Metadata and Validation System

## Status

Draft — after Commit A (Receipt infrastructure).

## Problem

The project has two different approaches to form validation and field behavior:

1. **Directories (Customers, Products):** simple `<select>` for organization, inline
   validation in Save handlers. No metadata beyond `Descriptor` (ui/display labels).

2. **Documents (Receipts):** multi-step lifecycle with cross-field dependencies
   (organization → customer → items), aggregation validation, and read-only after
   submission.

Documents introduce requirements that directories don't have:

- Fields depend on each other (changing org must clear customer and items).
- Validation must check not just individual fields but the aggregate as a whole.
- Entity selection (customer, product) needs a dedicated picker, not a `<select>`.
- UI state must be preserved on validation failure.
- Error messages must be specific to fields or the document as a whole.

Without a design, each new document type will reinvent its own validation and
field metadata — leading to inconsistency.

## Solution

Four interconnected subsystems are needed:

### 1. Lookup Mechanism (Entity Picker)

Large directories (customers, products) cannot use `<select>` with hundreds of
options. Instead, a dedicated **Lookup** mechanism is used.

```
Field label: Customer
Current value: Иванов И.И.    [Выбрать]
```

The `[Выбрать]` button navigates to the entity list page, which acts as a
**picker** when invoked in selection mode:

1. User clicks `[Выбрать]` on a document.
2. Browser navigates to `/organizations/{oid}/customers?mode=picker`.
3. User taps a row → row has a picker URL with return parameter:
   `/receipts/new?customer_id={id}&return_to=/receipts/new`
4. Server renders the document form with the selected customer pre-filled.

**Navigation contract:**

- A `mode=picker` query parameter switches the list page to picker mode
  (rows become selectable, no "New" button, no actions column).
- Each row in picker mode links back to the origin form with the selected
  entity ID and a `return_to` parameter.
- The origin handler reads the selection and `return_to` from query params,
  merges into the form state, and redirects or re-renders.

**Rules:**

- If no organization is selected, `[Выбрать]` must be disabled (no
  context to filter by).
- The picker URL must include `organization_id` so the list shows only
  entities belonging to that organization.
- This mechanism will be used for customers, products, and any future
  large directories.

### 2. Document Lifecycle

Documents have three distinct phases:

```
Unsaved (transient)
    │
    ▼
Validated
    │
    ▼
Persisted (in DB)
```

**Unsaved (Transient):** The document exists only in the user's browser.
No database record. The user can freely change organization, customer,
items, or abandon the form. No validation is applied. The form is purely
client-side with no server state.

**Validated:** The user clicks "Save". The server validates:

- `OrganizationID != 0`
- `CustomerID != 0`
- `len(Items) > 0`

If validation fails, the server re-renders the form with:

- All user-entered values preserved
- Error messages next to failed fields
- Aggregate errors (e.g., "Add at least one item") in a document-level
  error area

If validation passes, the document is saved to DB.

**Persisted:** The document exists in the database. It can be edited
(depending on status) or deleted. After submission (`SentAt != nil`),
it becomes read-only.

### 3. Field Metadata

The existing `entity.Field` struct needs to be extended from:

```go
type Field struct {
    GoName   string
    DBName   string
    Label    string
    Order    int
    List     bool
    Search   bool
    ReadOnly bool
}
```

To:

```go
type Field struct {
    GoName       string
    DBName       string
    Label        string
    Order        int
    List         bool
    Search       bool
    ReadOnly     bool
    Visible      bool         // show in UI (default true)
    Required     bool         // field-level validation
    DefaultValue string       // static default
    Validator    Validator    // field-level validation function
    DependsOn    []string     // fields this field depends on (GoNames)
}
```

**Validator interface:**

```go
type Validator func(value any) error
```

Return nil or an error message string.

**Rule:** `Required` is a shorthand for `Validator(func(v any) error {
if v == nil || v == "" || v == 0 { return errors.New("обязательное поле") }
return nil })`.

### 4. Dependency Graph

Field dependencies form a directed acyclic graph (DAG):

```
Organization
    │
    ▼
Customer
    │
    ▼
Items
```

When `Organization` changes:

- All fields that depend on it (`Customer`, `Items`) must be cleared.
- Their `DependsOn` lists contain `"OrganizationID"`.

This is declared in the field metadata, not in handler code:

```go
// Field: CustomerID
DependsOn: []string{"OrganizationID"},
```

The server (and eventually the client) walks the dependency graph on change:

1. User changes Organization.
2. Server finds all fields where `DependsOn` contains `OrganizationID`.
3. Those fields are cleared (set to zero value / empty).
4. The form is re-rendered with cleared fields.

**Implementation rules:**

- `DependsOn` is a flat list of GoNames. No complex expressions.
- If A depends on B, and B depends on C, changing C clears both B and A
  (transitive closure of the dependency graph).
- The graph is defined in the model's `Descriptor` registration, not in
  handlers.
- Initially, dependency clearing is server-side only (re-render on POST).
  Client-side clearing (JavaScript) is a future optimization.

## Impact on existing code

### Immediate (before Commit B)

- `Descriptor` metadata is extended with `Visible`, `Required`, `DependsOn`.
- Existing models (Customer, Product, Organization, User) are updated with
  new field metadata (backwards compatible — zero values mean "no constraint").

### Commit B (Receipt editor)

- Receipt descriptor declares field-level metadata.
- Picker mechanism is implemented for customer and product selection.
- Validation logic is extracted from handlers into a shared function.
- Dependency clearing is implemented in the receipt save handler.

### Future

- Picker mechanism is reused for all entity selections.
- Validation is extracted into `internal/entity` or a new `internal/validation`
  package.
- Client-side field clearing via JavaScript (HTMX or similar).

## Open questions

1. Should the picker return parameter be `return_to` or a dedicated
   `picker_return_url`? Using `return_to` is simpler but could conflict
   with other redirect logic.

2. Should dependency clearing happen via JavaScript progressively, or
   is server-side re-render on form POST sufficient for v1?

3. Should `Validator` be defined as an interface in Go or as a function
   type? Function type is simpler: `type Validator func(value any) error`.

## References

- `RFC-015-entity-descriptor.md` — existing field metadata
- `ARCHITECTURE.md` — section 11 (Entity Descriptor)
- `FUNCTIONAL_ORDERS.md` — sections 14-16 (lifecycle, metadata, validation)
