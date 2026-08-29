import type { Histogram } from "./stats.ts"
import { histogram } from "./stats.ts"

export type Report = {
  readonly url: string
  readonly connections: number
  readonly duration: number
  readonly errors: number
  readonly timeouts: number
  readonly non2xx: number
  readonly "2xx": number
  readonly latency: Histogram
  readonly requests: { readonly total: number }
  readonly throughput: { readonly total: number }
}

export type Totals = {
  readonly latencies: ReadonlyArray<number>
  readonly bytes: number
  readonly status2xx: number
  readonly non2xx: number
  readonly errors: number
  readonly timeouts: number
}

export const reportFrom = (
  options: {
    readonly url: URL
    readonly connections: number
    readonly durationSeconds: number
  },
  totals: Totals,
): Report => ({
  url: options.url.href,
  connections: options.connections,
  duration: options.durationSeconds,
  errors: totals.errors,
  timeouts: totals.timeouts,
  non2xx: totals.non2xx,
  "2xx": totals.status2xx,
  latency: histogram(totals.latencies),
  requests: {
    total: totals.status2xx + totals.non2xx + totals.errors + totals.timeouts,
  },
  throughput: { total: totals.bytes },
})

const fmt = (value: number): string => {
  if (Number.isInteger(value)) {
    return String(value)
  }
  return value.toFixed(2)
}

export const formatText = (report: Report, printLatency: boolean): string => {
  const lines = [
    `Running ${fmt(report.duration)}s test @ ${report.url}`,
    `${report.connections} connections`,
    "",
    `Stat     Avg     p50     p90     p99     Max`,
    `Latency  ${fmt(report.latency.average)}     ${
      fmt(report.latency.p50)
    }     ${fmt(report.latency.p90)}     ${fmt(report.latency.p99)}     ${
      fmt(report.latency.max)
    }`,
    "",
    `${report.requests.total} requests in ${
      fmt(report.duration)
    }s, ${report.throughput.total} bytes`,
    `${
      report["2xx"]
    } 2xx, ${report.non2xx} non2xx, ${report.errors} errors, ${report.timeouts} timeouts`,
  ]
  if (printLatency) {
    lines.push("")
    lines.push(
      `Latency min ${fmt(report.latency.min)} avg ${
        fmt(report.latency.average)
      } p50 ${fmt(report.latency.p50)} p90 ${fmt(report.latency.p90)} p99 ${
        fmt(report.latency.p99)
      } max ${fmt(report.latency.max)}`,
    )
  }
  return lines.join("\n")
}
