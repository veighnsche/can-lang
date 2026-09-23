# Small Can examples

Start here if you want short, runnable examples of current Can syntax. Each numbered
file contains one idea and attached `asserts` rows that supply inputs and expected
results. The gallery is one project so the compiler can check all examples together.

Use a qualified `canlc` installation (see the repository [README](../../README.md)):

```sh
canlc build examples/gallery
```

Run one assertion from the repository root with its full package and declaration
names. For example:

```sh
canlc assert examples/gallery can.project.root/gallery can.project.root/gallery::fizzbuzz both
```

`build` verifies every assertion before publishing. `assert` checks examples without
publishing. You can also run the CLI greeting (the gallery entry point):

```sh
canlc run examples/gallery -- Ada
# prints: Hello, Ada!
```

The other lessons run through their assertion rows, which provide concrete inputs
and expected results.

| File | What it teaches | Main function to inspect |
| --- | --- | --- |
| [01-hello-world.can](src/01-hello-world.can) | First function and assertion | `hello` |
| [02-cli-argument.can](src/02-cli-argument.can) | CLI style `str[]` arguments | `greet_argument` |
| [03-arithmetic.can](src/03-arithmetic.can) | Integer operators | `arithmetic` |
| [04-temperature.can](src/04-temperature.can) | Floating point conversion | `celsius_to_fahrenheit` |
| [05-leap-year.can](src/05-leap-year.can) | Boolean tests and calendar rules | `leap_year` |
| [06-fizzbuzz.can](src/06-fizzbuzz.can) | Ordered nested matches | `fizzbuzz` |
| [07-fibonacci.can](src/07-fibonacci.can) | Simple recursion | `fibonacci` |
| [08-factorial.can](src/08-factorial.can) | Base case and recursive step | `factorial` |
| [09-gcd.can](src/09-gcd.can) | Euclid's algorithm and `relay` | `gcd` |
| [10-prime-check.can](src/10-prime-check.can) | Trial divisors | `is_prime` |
| [11-palindrome.can](src/11-palindrome.can) | Recursive string comparison (ASCII) | `palindrome` |
| [12-word-character-count.can](src/12-word-character-count.can) | Splitting and Unicode scalar counts | `count_text` |
| [13-boolean-match.can](src/13-boolean-match.can) | `false` before `true` in a match | `describe_boolean` |
| [14-record-variant-match.can](src/14-record-variant-match.can) | Records inside a variant | `area_units` |
| [15-exhaustive-match.can](src/15-exhaustive-match.can) | Covering every variant case | `color_name` |
| [16-optional-values.can](src/16-optional-values.can) | Explicit presence and absence | `first_word` |
| [17-named-error-recovery.can](src/17-named-error-recovery.can) | Recovering a named error | `is_positive` |
| [18-relay.can](src/18-relay.can) | Forwarding a result or error | `require_positive` |
| [19-immutable-record-update.can](src/19-immutable-record-update.can) | Copying a record with `with` | `increment` |
| [20-array-map.can](src/20-array-map.can) | Transforming an array | `doubled` |
| [21-array-filter.can](src/21-array-filter.can) | Selecting array items | `evens` |
| [22-array-fold.can](src/22-array-fold.can) | Accumulating an array | `sum` |
| [23-array-sort.can](src/23-array-sort.can) | Sorting a copied array | `sorted` |
| [24-map-word-frequencies.can](src/24-map-word-frequencies.can) | Immutable map and word frequencies | `frequency` |
| [25-generic-function.can](src/25-generic-function.can) | A type parameter | `identity` |
| [26-callable-argument.can](src/26-callable-argument.can) | Passing a callable | `apply` |
| [27-recursion-vs-collection.can](src/27-recursion-vs-collection.can) | Recursion beside `fold` | `recursive_sum`, `folded_sum` |
| [28-attached-assertions.can](src/28-attached-assertions.can) | Several named test cases | `square` |

[Deliberate failing assertion](../../tests/integration/testdata/gallery-failing/README.md)
is a separate project outside the maintained examples. It demonstrates the
diagnostic from a wrong expected result.
