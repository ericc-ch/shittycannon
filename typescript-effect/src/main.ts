import { DenoRuntime, DenoServices } from "@effect/platform-deno"
import * as DenoHttpClient from "@effect/platform-deno/DenoHttpClient"
import { Effect, Layer } from "effect"
import { Command } from "effect/unstable/cli"
import { shittycannon } from "./command.ts"

const KeepAlive = true

const MainLive = Layer.mergeAll(
  DenoServices.layer,
  DenoHttpClient.layer.pipe(
    Layer.provide(
      Layer.succeed(DenoHttpClient.RequestInit, { keepalive: KeepAlive }),
    ),
  ),
)

DenoRuntime.runMain(
  Command.run(shittycannon, { version: "0.0.0" }).pipe(
    Effect.provide(MainLive),
  ),
)
