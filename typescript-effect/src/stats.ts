export type Histogram = {
  readonly min: number
  readonly max: number
  readonly average: number
  readonly p50: number
  readonly p90: number
  readonly p99: number
}

export const emptyHistogram: Histogram = {
  min: 0,
  max: 0,
  average: 0,
  p50: 0,
  p90: 0,
  p99: 0,
}

const percentile = (
  sorted: ReadonlyArray<number>,
  p: number,
): number => {
  if (sorted.length === 0) {
    return 0
  }
  const rank = Math.ceil((p / 100) * sorted.length) - 1
  const index = Math.min(sorted.length - 1, Math.max(0, rank))
  const value = sorted[index]
  if (value === undefined) {
    return 0
  }
  return value
}

export const histogram = (samples: ReadonlyArray<number>): Histogram => {
  if (samples.length === 0) {
    return emptyHistogram
  }
  const sorted = samples.toSorted((left, right) => left - right)
  const first = sorted[0]
  const last = sorted[sorted.length - 1]
  if (first === undefined || last === undefined) {
    return emptyHistogram
  }
  const total = sorted.reduce((sum, sample) => sum + sample, 0)
  return {
    min: first,
    max: last,
    average: total / sorted.length,
    p50: percentile(sorted, 50),
    p90: percentile(sorted, 90),
    p99: percentile(sorted, 99),
  }
}
