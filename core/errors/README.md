# core/errors

Standard HTTP error contract for the SpoolIQ API.

## Wire format

Every error response uses the same JSON envelope:

```json
{
  "error":   "Mensagem em pt-BR",
  "message": "Mensagem em pt-BR",
  "code":    "snake_case_code",
  "fields":  { "campo": "mensagem" }
}
```

- `error` and `message` carry the **same** pt-BR text. `error` exists for
  backward compatibility; `message` is the forward name.
- `code` is a stable, machine-readable `snake_case` string.
- `fields` is only present for validation errors (omitted otherwise).

## Returning errors from handlers

Build an `APIError` with a constructor and hand it to `errors.Respond` (or
`errors.AbortWith` inside middleware):

```go
func (uc *UseCase) Get(c *gin.Context) {
    id, err := uuid.Parse(c.Param("id"))
    if err != nil {
        errors.Respond(c, errors.BadRequest("invalid_customer_id", "ID de cliente inválido"))
        return
    }

    customer, err := uc.repo.FindByID(c.Request.Context(), id)
    if err != nil {
        // gorm.ErrRecordNotFound is mapped to 404 not_found automatically.
        errors.Respond(c, err)
        return
    }

    c.JSON(http.StatusOK, customer)
}
```

### Constructors

| Constructor                       | Status | Use for                         |
|-----------------------------------|--------|---------------------------------|
| `BadRequest(code, msg)`           | 400    | malformed/invalid input         |
| `Unauthorized(code, msg)`         | 401    | missing/invalid auth            |
| `PaymentRequired(code, msg)`      | 402    | subscription/payment gating     |
| `Forbidden(code, msg)`            | 403    | authenticated but not allowed   |
| `NotFoundErr(code, msg)`          | 404    | resource not found              |
| `Conflict(code, msg)`             | 409    | uniqueness / state conflict     |
| `Validation(fields)`              | 400    | per-field validation failures   |
| `Internal()`                      | 500    | generic failure (no detail)     |

> Note: `NotFoundErr` is named with the `Err` suffix to avoid colliding with the
> legacy `NotFound(msg) *AppError` helper, which still exists.

## Mapping precedence (`Respond` / `AbortWith`)

`Respond` inspects the error in this order:

1. `*APIError` — used as-is.
2. `*AppError` (legacy) — its `HTTPStatus()` with a code derived from its type;
   the pt-BR message is preserved.
3. `gorm.ErrRecordNotFound` — `404 not_found`.
4. `validator.ValidationErrors` — `400 validation_error` with pt-BR `fields`.
5. JSON/binding decode errors — `400 invalid_request`.
6. anything else — `500 internal_error`. The real error is **logged** (via the
   logger wired with `errors.SetLogger`) and **never** written to the body.

## Validation

Validate request/DTO structs with the shared validator in
[`core/validation`](../validation):

```go
if err := validation.Validate(req); err != nil {
    errors.Respond(c, err) // -> 400 validation_error with pt-BR fields
    return
}
```

Field names in the response come from the `json` tag. The shared validator is
also registered as gin's binding validator, so `binding:"required"` tags on
structs passed to `c.ShouldBind*` are enforced too, and both produce the same
envelope.

## Migration guide for feature engineers

When adopting this contract in a feature (separate PRs):

1. Replace ad-hoc `c.JSON(status, gin.H{"error": ...})` with an `APIError`
   constructor + `errors.Respond(c, err)`.
2. Replace per-usecase `validator.New()` instances with
   `validation.Validate(...)`.
3. Prefer specific, stable `code` values (e.g. `invalid_customer_id`,
   `email_already_exists`) so clients can branch on them.
4. Keep user-facing messages in pt-BR.
5. Let repository `gorm.ErrRecordNotFound` bubble up — `Respond` turns it into a
   clean `404 not_found` for you.
