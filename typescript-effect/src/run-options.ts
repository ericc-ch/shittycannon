import { Data } from "effect"
import type { HttpMethod } from "effect/unstable/http/HttpMethod"

export type Stop = Data.TaggedEnum<{
  Duration: { readonly seconds: number }
  Amount: { readonly requests: number }
}>

export const Stop = Data.taggedEnum<Stop>()

export type Body = Data.TaggedEnum<{
  Empty: { readonly empty?: never }
  Text: { readonly value: string }
}>

export const Body = Data.taggedEnum<Body>()

export type RunOptions = {
  readonly url: URL
  readonly connections: number
  readonly stop: Stop
  readonly method: HttpMethod
  readonly headers: Record<string, string>
  readonly body: Body
  readonly timeoutSeconds: number
}
