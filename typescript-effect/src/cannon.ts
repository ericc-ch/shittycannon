import { Data, Duration, Effect, Fiber, Match, Option, Ref } from "effect"
import * as HttpClient from "effect/unstable/http/HttpClient"
import * as HttpClientError from "effect/unstable/http/HttpClientError"
import * as HttpClientRequest from "effect/unstable/http/HttpClientRequest"
import { hasBody } from "effect/unstable/http/HttpMethod"
import { reportFrom } from "./report.ts"
import type { RunOptions } from "./run-options.ts"

const AllowRequest = true
const DenyRequest = false

type TotalsState = {
  latencies: Array<number>
  bytes: number
  status2xx: number
  non2xx: number
  errors: number
  timeouts: number
}

type Intake = Data.TaggedEnum<{
  Open: { readonly remaining: Option.Option<number> }
  Closed: { readonly closed?: never }
}>

const Intake = Data.taggedEnum<Intake>()

const fireOnce = Effect.fnUntraced(function* (
  options: RunOptions,
  totals: Ref.Ref<TotalsState>,
) {
  const client = yield* HttpClient.HttpClient
  const base = HttpClientRequest.setHeaders(
    HttpClientRequest.make(options.method)(options.url),
    options.headers,
  )
  const request = Match.value(options.body).pipe(
    Match.tagsExhaustive({
      Empty: () => base,
      Text: (body) =>
        hasBody(options.method)
          ? HttpClientRequest.bodyText(base, body.value)
          : base,
    }),
  )

  yield* Effect.gen(function* () {
    const [elapsed, { response, buffer }] = yield* Effect.timed(
      Effect.gen(function* () {
        const response = yield* client.execute(request)
        const buffer = yield* response.arrayBuffer
        return { response, buffer }
      }),
    )
    yield* Ref.update(totals, (current) => {
      current.latencies.push(Duration.toMillis(elapsed))
      current.bytes += buffer.byteLength
      if (response.status >= 200 && response.status < 300) {
        current.status2xx += 1
      } else {
        current.non2xx += 1
      }
      return current
    })
  }).pipe(
    Effect.timeout(Duration.seconds(options.timeoutSeconds)),
    Effect.catchTag(
      "TimeoutError",
      () =>
        Ref.update(totals, (current) => {
          current.timeouts += 1
          return current
        }),
    ),
    Effect.catchIf(
      HttpClientError.isHttpClientError,
      () =>
        Ref.update(totals, (current) => {
          current.errors += 1
          return current
        }),
    ),
  )
})

const connectionLoop = Effect.fnUntraced(function* (
  options: RunOptions,
  totals: Ref.Ref<TotalsState>,
  intake: Ref.Ref<Intake>,
) {
  while (true) {
    const allowed = yield* Ref.modify(
      intake,
      (state) =>
        Match.value(state).pipe(
          Match.tagsExhaustive({
            Closed: () => [DenyRequest, state] as const,
            Open: (open) =>
              Option.match(open.remaining, {
                onNone: () => [AllowRequest, state] as const,
                onSome: (count) => {
                  if (count <= 0) {
                    return [DenyRequest, Intake.Closed({})] as const
                  }
                  return [
                    AllowRequest,
                    Intake.Open({ remaining: Option.some(count - 1) }),
                  ] as const
                },
              }),
          }),
        ),
    )

    if (!allowed) {
      return
    }

    yield* fireOnce(options, totals)
  }
})

export const runCannon = Effect.fn("runCannon")(
  function* (options: RunOptions) {
    const totals = yield* Ref.make({
      latencies: [] as Array<number>,
      bytes: 0,
      status2xx: 0,
      non2xx: 0,
      errors: 0,
      timeouts: 0,
    })

    const intake = yield* Ref.make<Intake>(
      Match.value(options.stop).pipe(
        Match.tagsExhaustive({
          Amount: (amount) =>
            Intake.Open({ remaining: Option.some(amount.requests) }),
          Duration: () => Intake.Open({ remaining: Option.none() }),
        }),
      ),
    )

    const workers = Effect.all(
      Array.from(
        { length: options.connections },
        () => connectionLoop(options, totals, intake),
      ),
      { concurrency: "unbounded" },
    )

    const [elapsed, snapshot] = yield* Effect.timed(
      Effect.gen(function* () {
        const fiber = yield* Effect.forkChild(workers)
        yield* Match.value(options.stop).pipe(
          Match.tagsExhaustive({
            Duration: (duration) =>
              Effect.gen(function* () {
                yield* Effect.sleep(Duration.seconds(duration.seconds))
                yield* Ref.set(intake, Intake.Closed({}))
                yield* Fiber.join(fiber)
              }),
            Amount: () => Fiber.join(fiber),
          }),
        )
        return yield* Ref.get(totals)
      }),
    )

    return reportFrom(
      {
        url: options.url,
        connections: options.connections,
        durationSeconds: Duration.toSeconds(elapsed),
      },
      snapshot,
    )
  },
)
