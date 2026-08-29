TypeScript + Effect implementation of the shittycannon autocannon subset.

Use Deno. Run first-party `.ts` with `deno run`. Tests live in `test/` as
`Deno.test()` via `deno test`. Format with `deno fmt`. Lint with `deno lint`.

Effect CLI lives at `effect/unstable/cli`. HTTP and files go through
`@effect/platform-deno`.

After completing a task, run `deno task check` (typecheck, test, lint, fmt
--check). Run the CLI with `deno task start -- <args>`.

For TypeScript style, follow the code-conventions skill.
