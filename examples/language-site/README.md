# Can language site prototype

This small site is authored in Can. It serves a landing page at `/` and a
getting-started page at `/docs`. The HTML uses Can's typed safe constructors;
the stylesheet is a declared asset. This is a public-facing prototype, while
the repository's `docs/` tree remains the detailed design and implementation
record.

From the repository root with a qualified `canlc` installation:

```sh
canlc build examples/language-site
canlc run examples/language-site
```

Open <http://127.0.0.1:18487/> and <http://127.0.0.1:18487/docs>. The
prototype binds to loopback on port 18487 and serves until interrupted.
