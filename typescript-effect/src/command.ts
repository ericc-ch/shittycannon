import { Console, Effect, FileSystem, Option, Schema } from "effect"
import { Argument, CliError, Command, Flag } from "effect/unstable/cli"
import { runCannon } from "./cannon.ts"
import { formatText } from "./report.ts"
import { Body, type RunOptions, Stop } from "./run-options.ts"

const Disabled = false
const DefaultConnections = 10
const DefaultDuration = 10
const DefaultTimeout = 10
const DefaultMethod = "GET"

const PositiveInt = Schema.Int.check(Schema.isGreaterThan(0))

export const shittycannon = Command.make(
  "shittycannon",
  {
    url: Argument.string("url").pipe(
      Argument.withDescription("HTTP or HTTPS URL to benchmark"),
      Argument.mapEffect((input) =>
        Schema.decodeUnknownEffect(
          Schema.URLFromString.check(
            Schema.makeFilter((url: URL) => {
              if (url.protocol === "http:" || url.protocol === "https:") {
                return true
              }
              return "URL must use http or https"
            }),
          ),
        )(input).pipe(
          Effect.mapError((error) =>
            new CliError.InvalidValue({
              option: "url",
              value: input,
              expected: error.message,
              kind: "argument",
            })
          ),
        )
      ),
    ),
    connections: Flag.integer("connections").pipe(
      Flag.withAlias("c"),
      Flag.withDescription(
        "The number of concurrent connections to use. default: 10.",
      ),
      Flag.withDefault(DefaultConnections),
      Flag.withSchema(PositiveInt),
    ),
    duration: Flag.integer("duration").pipe(
      Flag.withAlias("d"),
      Flag.withDescription(
        "The number of seconds to run the autocannon. default: 10.",
      ),
      Flag.withDefault(DefaultDuration),
      Flag.withSchema(PositiveInt),
    ),
    amount: Flag.integer("amount").pipe(
      Flag.withAlias("a"),
      Flag.withDescription(
        "The number of requests to make before exiting the benchmark. If set, duration is ignored.",
      ),
      Flag.withSchema(PositiveInt),
      Flag.optional,
    ),
    method: Flag.string("method").pipe(
      Flag.withAlias("m"),
      Flag.withDescription("The HTTP method to use. default: 'GET'."),
      Flag.withDefault(DefaultMethod),
      Flag.mapEffect((input) =>
        Schema.decodeUnknownEffect(
          Schema.Literals([
            "GET",
            "POST",
            "PUT",
            "DELETE",
            "PATCH",
            "HEAD",
            "OPTIONS",
            "TRACE",
          ]),
        )(input.toUpperCase()).pipe(
          Effect.mapError((error) =>
            new CliError.InvalidValue({
              option: "method",
              value: input,
              expected: error.message,
              kind: "flag",
            })
          ),
        )
      ),
    ),
    timeout: Flag.integer("timeout").pipe(
      Flag.withAlias("t"),
      Flag.withDescription(
        "The number of seconds before timing out and resetting a connection. default: 10",
      ),
      Flag.withDefault(DefaultTimeout),
      Flag.withSchema(PositiveInt),
    ),
    body: Flag.string("body").pipe(
      Flag.withAlias("b"),
      Flag.withDescription("The body of the request."),
      Flag.optional,
    ),
    input: Flag.file("input", { mustExist: true }).pipe(
      Flag.withAlias("i"),
      Flag.withDescription(
        "The body of the request. See '-b/body' for more details.",
      ),
      Flag.optional,
    ),
    headers: Flag.keyValuePair("headers").pipe(
      Flag.withAlias("H"),
      Flag.withDescription("The request headers."),
      Flag.withDefault({}),
    ),
    json: Flag.boolean("json").pipe(
      Flag.withAlias("j"),
      Flag.withDescription(
        "Print the output as newline delimited JSON. This will cause the progress bar and results not to be rendered. default: false.",
      ),
      Flag.withDefault(Disabled),
    ),
    latency: Flag.boolean("latency").pipe(
      Flag.withAlias("l"),
      Flag.withDescription("Print all the latency data. default: false."),
      Flag.withDefault(Disabled),
    ),
  },
  Effect.fn("shittycannon")(function* (config) {
    if (Option.isSome(Option.all([config.body, config.input]))) {
      return yield* new CliError.UserError({
        cause: "use either -b/--body or -i/--input, not both",
        userMessage: "use either -b/--body or -i/--input, not both",
      })
    }
    const runOptions: RunOptions = {
      url: config.url,
      connections: config.connections,
      stop: Option.match(config.amount, {
        onNone: () => Stop.Duration({ seconds: config.duration }),
        onSome: (requests) => Stop.Amount({ requests }),
      }),
      method: config.method,
      headers: config.headers,
      body: yield* Option.match(config.body, {
        onSome: (value) => Effect.succeed(Body.Text({ value })),
        onNone: () =>
          Option.match(config.input, {
            onSome: (path) =>
              Effect.gen(function* () {
                const fs = yield* FileSystem.FileSystem
                const value = yield* fs.readFileString(path)
                return Body.Text({ value })
              }),
            onNone: () => Effect.succeed(Body.Empty({})),
          }),
      }),
      timeoutSeconds: config.timeout,
    }
    const report = yield* runCannon(runOptions)
    if (config.json) {
      yield* Console.log(JSON.stringify(report))
      return
    }
    yield* Console.log(formatText(report, config.latency))
  }),
).pipe(
  Command.withDescription("HTTP/1 load tester (autocannon subset)"),
)
