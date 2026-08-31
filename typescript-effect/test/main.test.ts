import {
  assertEquals,
  assertNotEquals,
  assertStringIncludes,
} from "@std/assert"

const runCli = async (
  args: ReadonlyArray<string>,
): Promise<{ code: number; stdout: string; stderr: string }> => {
  const command = new Deno.Command("deno", {
    args: ["run", "-A", "src/main.ts", ...args],
    cwd: new URL("..", import.meta.url).pathname,
    stdout: "piped",
    stderr: "piped",
  })
  const output = await command.output()
  const decoder = new TextDecoder()
  return {
    code: output.code,
    stdout: decoder.decode(output.stdout),
    stderr: decoder.decode(output.stderr),
  }
}

Deno.test("--help lists the subset flags", async () => {
  const result = await runCli(["--help"])
  assertEquals(result.code, 0)
  assertStringIncludes(result.stdout, "--connections")
  assertStringIncludes(result.stdout, "--duration")
  assertStringIncludes(result.stdout, "--amount")
  assertStringIncludes(result.stdout, "--method")
  assertStringIncludes(result.stdout, "--headers")
  assertStringIncludes(result.stdout, "--body")
  assertStringIncludes(result.stdout, "--input")
  assertStringIncludes(result.stdout, "--timeout")
  assertStringIncludes(result.stdout, "--json")
  assertStringIncludes(result.stdout, "--latency")
})

Deno.test("unknown autocannon flags fail", async () => {
  const result = await runCli(["--pipelining", "1", "http://127.0.0.1/"])
  assertNotEquals(result.code, 0)
})

const serveLocal = (
  handler: (request: Request) => Response | Promise<Response>,
): Deno.HttpServer<Deno.NetAddr> => {
  return Deno.serve(
    {
      hostname: "127.0.0.1",
      port: 0,
      onListen: () => undefined,
    },
    (request: Request): Promise<Response> => Promise.resolve(handler(request)),
  )
}

const tcpPort = (server: Deno.HttpServer<Deno.NetAddr>): number => {
  const addr = server.addr
  if (!("port" in addr)) {
    throw new Error("expected a TCP address")
  }
  return addr.port
}

Deno.test("amount run against a local server", async () => {
  let hits = 0
  const server = serveLocal(() => {
    hits += 1
    return new Response("ok")
  })
  try {
    const result = await runCli([
      "-c",
      "2",
      "-a",
      "20",
      "-j",
      `http://127.0.0.1:${tcpPort(server)}/`,
    ])
    assertEquals(result.code, 0, result.stderr)
    const report = JSON.parse(result.stdout) as {
      requests: { total: number }
      "2xx": number
    }
    assertEquals(report.requests.total, 20)
    assertEquals(report["2xx"], 20)
    assertEquals(hits, 20)
  } finally {
    await server.shutdown()
  }
})

Deno.test("POST body from -i file", async () => {
  const seen: Array<{ method: string; body: string }> = []
  const server = serveLocal(async (request) => {
    seen.push({
      method: request.method,
      body: await request.text(),
    })
    return new Response("ok")
  })
  const bodyFile = await Deno.makeTempFile()
  await Deno.writeTextFile(bodyFile, '{"ok":true}')
  try {
    const result = await runCli([
      "-c",
      "1",
      "-a",
      "3",
      "-m",
      "POST",
      "-i",
      bodyFile,
      "-H",
      "content-type=application/json",
      "-j",
      `http://127.0.0.1:${tcpPort(server)}/`,
    ])
    assertEquals(result.code, 0, result.stderr)
    assertEquals(seen, [
      { method: "POST", body: '{"ok":true}' },
      { method: "POST", body: '{"ok":true}' },
      { method: "POST", body: '{"ok":true}' },
    ])
  } finally {
    await server.shutdown()
    await Deno.remove(bodyFile)
  }
})

Deno.test("-b and -i together is a user error", async () => {
  const bodyFile = await Deno.makeTempFile()
  await Deno.writeTextFile(bodyFile, "x")
  try {
    const result = await runCli([
      "-b",
      "hello",
      "-i",
      bodyFile,
      "http://127.0.0.1/",
    ])
    assertNotEquals(result.code, 0)
    assertStringIncludes(`${result.stdout}${result.stderr}`, "either")
  } finally {
    await Deno.remove(bodyFile)
  }
})

Deno.test("non-2xx responses are counted", async () => {
  const server = serveLocal(() => new Response("nope", { status: 500 }))
  try {
    const result = await runCli([
      "-c",
      "1",
      "-a",
      "4",
      "-j",
      `http://127.0.0.1:${tcpPort(server)}/`,
    ])
    assertEquals(result.code, 0, result.stderr)
    const report = JSON.parse(result.stdout) as {
      "2xx": number
      non2xx: number
    }
    assertEquals(report["2xx"], 0)
    assertEquals(report.non2xx, 4)
  } finally {
    await server.shutdown()
  }
})

Deno.test("timeouts are counted when the handler is slow", async () => {
  const server = serveLocal(async () => {
    await new Promise((resolve) => setTimeout(resolve, 3000))
    return new Response("ok")
  })
  try {
    const result = await runCli([
      "-c",
      "1",
      "-a",
      "1",
      "-t",
      "1",
      "-j",
      `http://127.0.0.1:${tcpPort(server)}/`,
    ])
    assertEquals(result.code, 0, result.stderr)
    const report = JSON.parse(result.stdout) as { timeouts: number }
    assertEquals(report.timeouts, 1)
  } finally {
    await server.shutdown()
  }
})

Deno.test("duration counts the in-flight request", async () => {
  const server = serveLocal(async () => {
    await new Promise((resolve) => setTimeout(resolve, 1500))
    return new Response("ok")
  })
  try {
    const result = await runCli([
      "-c",
      "1",
      "-d",
      "1",
      "-t",
      "5",
      "-j",
      `http://127.0.0.1:${tcpPort(server)}/`,
    ])
    assertEquals(result.code, 0, result.stderr)
    const report = JSON.parse(result.stdout) as {
      "2xx": number
      requests: { total: number }
    }
    assertEquals(report["2xx"], 1)
    assertEquals(report.requests.total, 1)
  } finally {
    await server.shutdown()
  }
})
