# Terms

**shittycannon** is a practice reimplementation of a subset of [autocannon](https://github.com/mcollina/autocannon), in multiple languages. The first implementation is TypeScript with Effect in `typescript-effect/`, using Deno.

A **connection** is one concurrent HTTP client that holds at most one in-flight request at a time. Pipelining is out of this subset.

**Duration** (`-d`) is wall-clock seconds to run. **Amount** (`-a`) is a request count cap; if set, duration is ignored.

**CLI shape** means the included flags, defaults, and names match autocannon. Unsupported autocannon flags are not part of this subset.
