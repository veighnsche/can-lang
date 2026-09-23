# Deliberately failing assertion

This project is separate from the passing examples. `square(3)` returns `9`, while
its `deliberately_wrong` assertion expects `8`. Run it to see the compiler's
outcome mismatch and a nonzero exit status:

```sh
canlc build tests/integration/testdata/gallery-failing
```

Do not include this project in the normal gallery check. The passing examples are
verified with `canlc build examples/gallery`.
