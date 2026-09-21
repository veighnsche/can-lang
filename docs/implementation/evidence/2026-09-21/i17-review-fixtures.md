# I17 review correction: raw provider fixture admission

Independent review found that an invalid response status could fail during exchange construction after provider evidence had been recorded. Handling that ordinary failure could incorrectly pass an assertion.

Registration now copies and validates all response configurations before installing the fixture table. Status must be an integer from 200 through 599; 204, 205 and 304 require an empty body. Native Headers/Response construction validates headers. Admission failures record a sticky malformed-fixture violation. Consumption records evidence only after successful response construction. Malformed JSON payloads remain supported for decoder negative tests.

Regression covers status 999, invalid header names, and nonempty bodies for all three null-body statuses. Catching registration errors still produces a failed assertion without provider evidence.

Validation: Bun 1.4.2, `bun test runtime/test`: 95 pass, 0 fail, 723 expectations. `git diff --check` passes.
