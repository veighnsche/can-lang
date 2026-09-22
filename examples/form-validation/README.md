# form-validation

The admission example for typed forms and 422 feedback: a signup page
whose submit button disables during the request, whose progress element
is named through `htmx::indicator_id`, and whose result region swaps
either a validated fragment or a 422 validation fragment.

## Layout

- `src/records/` — `signup_form` (one name, repeated `tags`, optional
  `note`) and `signup_view`.
- `src/model/` — `validate_signup`, the ordinary form-to-view mapping.
- `src/render/` — the signup page plus safe result fragments.
- `src/web/` — the two callbacks, route mounting, `boot`, and the
  serving `main`.
- `assets/site.css` — the only declared static asset.

## Run

```
canlc assert examples/form-validation
canlc build examples/form-validation
```

`main` takes one port argument. No database is involved.

## Behavior

- `GET /signup` serves the page; the form posts to
  `/signup/validate` and swaps into `#signup_results`.
- Blank names, body-limit breaches, malformed requests, and undecodable
  bodies answer 422 with a safe fragment; valid forms answer 200 with
  the name and tag count. Repeated `tags` values decode into the array
  and a missing `note` decodes to none.
- Unknown routes answer compiler-owned 404/405 without callbacks.
- SIGINT/SIGTERM stops the server and exits zero.
