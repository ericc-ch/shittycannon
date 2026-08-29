import { assertAlmostEquals, assertEquals } from "@std/assert"
import { emptyHistogram, histogram } from "../src/stats.ts"

Deno.test("histogram is empty when there are no samples", () => {
  assertEquals(histogram([]), emptyHistogram)
})

Deno.test("histogram uses the only sample for every stat", () => {
  assertEquals(histogram([12]), {
    min: 12,
    max: 12,
    average: 12,
    p50: 12,
    p90: 12,
    p99: 12,
  })
})

Deno.test("histogram nearest-rank percentiles on ten samples", () => {
  const samples = [1, 2, 3, 4, 5, 6, 7, 8, 9, 10]
  const result = histogram(samples)
  assertEquals(result.min, 1)
  assertEquals(result.max, 10)
  assertAlmostEquals(result.average, 5.5)
  assertEquals(result.p50, 5)
  assertEquals(result.p90, 9)
  assertEquals(result.p99, 10)
})
