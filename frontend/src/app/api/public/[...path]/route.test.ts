// @vitest-environment node
import { NextRequest } from "next/server";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

vi.mock("server-only", () => ({}));
vi.mock("@/lib/api/backendUrl", () => ({ BACKEND_INTERNAL_URL: "http://api:8080" }));

import { GET, POST } from "./route";

const params = (...path: string[]) => ({ params: Promise.resolve({ path }) });

describe("proxy BFF público (allowlist)", () => {
  let upstream: ReturnType<typeof vi.fn>;
  beforeEach(() => {
    upstream = vi.fn(
      async () =>
        new Response('{"data":1,"error":null}', {
          status: 200,
          headers: { "content-type": "application/json" },
        }),
    );
    vi.stubGlobal("fetch", upstream);
  });
  afterEach(() => {
    vi.unstubAllGlobals();
    vi.unstubAllEnvs();
  });

  it("sem proxy confiável, X-Forwarded-For do cliente NÃO é repassado (IP forjado burlava o limite)", async () => {
    await POST(
      new NextRequest("http://app/x", {
        method: "POST",
        body: "{}",
        headers: { "x-forwarded-for": "203.0.113.66" },
      }),
      params("v1", "contact", "messages"),
    );
    expect(upstream.mock.calls[0]![1].headers["X-Forwarded-For"]).toBeUndefined();
  });

  it("blog público passa; rota fora da allowlist, não", async () => {
    const ok = await GET(new NextRequest("http://app/api/public/v1/blog/posts"), params("v1", "blog", "posts"));
    expect(ok.status).toBe(200);
    expect(String(upstream.mock.calls[0]![0])).toBe("http://api:8080/api/v1/blog/posts");
    const fora = await GET(new NextRequest("http://app/api/public/v1/me"), params("v1", "me"));
    expect(fora.status).toBe(404);
    expect(upstream).toHaveBeenCalledTimes(1);
  });

  it("repassa o Range e devolve o 206 com o intervalo", async () => {
    upstream.mockResolvedValueOnce(
      new Response(new Uint8Array([37, 80, 68, 70]), {
        status: 206,
        headers: { "content-type": "application/pdf", "content-range": "bytes 0-3/100", "accept-ranges": "bytes", etag: '"abc"' },
      }),
    );
    const res = await GET(
      new NextRequest("http://app/api/public/v1/blog/posts/1/capa", { headers: { range: "bytes=0-3", "if-range": '"abc"' } }),
      params("v1", "blog", "posts", "1", "capa"),
    );
    expect(res.status).toBe(206);
    expect(res.headers.get("content-range")).toBe("bytes 0-3/100");
    expect(res.headers.get("accept-ranges")).toBe("bytes");
    expect(res.headers.get("etag")).toBe('"abc"');
    expect(new Uint8Array(await res.arrayBuffer())).toEqual(new Uint8Array([37, 80, 68, 70]));
    const init = upstream.mock.calls[0]![1];
    expect(init.headers.Range).toBe("bytes=0-3");
    expect(init.headers["If-Range"]).toBe('"abc"');
    // Range malformado ou If-Range enorme não passam.
    await GET(
      new NextRequest("http://app/x", { headers: { range: "bytes=0-1,5-9", "if-range": "x".repeat(201) } }),
      params("v1", "blog", "posts", "1", "capa"),
    );
    expect(upstream.mock.calls[1]![1].headers.Range).toBeUndefined();
    expect(upstream.mock.calls[1]![1].headers["If-Range"]).toBeUndefined();
  });

  it("encaminha rotas da allowlist, sem token, com o IP do visitante (atrás de proxy confiável)", async () => {

    vi.stubEnv("TRUST_PROXY_HEADERS", "true");
    const res = await GET(
      new NextRequest("http://app/api/public/v1/catalog/services?q=x", {
        headers: { "x-forwarded-for": "203.0.113.9" },
      }),
      params("v1", "catalog", "services"),
    );
    expect(res.status).toBe(200);
    const [url, init] = upstream.mock.calls[0]!;
    expect(String(url)).toBe("http://api:8080/api/v1/catalog/services?q=x");
    expect(init.headers.Authorization).toBeUndefined();
    expect(init.headers["X-Forwarded-For"]).toBe("203.0.113.9");
    expect(init.body).toBeUndefined();

    await POST(
      new NextRequest("http://app/x", {
        method: "POST",
        body: '{"a":1}',
        headers: { "x-request-id": "r1" },
      }),
      params("v1", "contact", "messages"),
    );
    const post = upstream.mock.calls[1]![1];
    expect(new TextDecoder().decode(post.body)).toBe('{"a":1}');
    expect(post.headers["X-Request-ID"]).toBe("r1");
    expect(post.headers["X-Forwarded-For"]).toBeUndefined();
  });

  it("sem proxy confiável, X-Forwarded-For do cliente NÃO é repassado (IP forjado burlava o limite)", async () => {
    await POST(
      new NextRequest("http://app/x", {
        method: "POST",
        body: "{}",
        headers: { "x-forwarded-for": "203.0.113.66" },
      }),
      params("v1", "contact", "messages"),
    );
    expect(upstream.mock.calls[0]![1].headers["X-Forwarded-For"]).toBeUndefined();
  });

  it("fora da allowlist (rota, método ou '..') responde 404", async () => {
    const cases: [string, string[]][] = [
      ["GET", ["v1", "me"]],
      ["POST", ["v1", "catalog", "services"]],
      ["GET", ["v1", "catalog", "..", "admin"]],
    ];
    for (const [method, path] of cases) {
      const req = new NextRequest("http://app/x", {
        method,
        body: method === "POST" ? "{}" : undefined,
      });
      const res = await (method === "GET" ? GET : POST)(req, params(...path));
      expect(res.status).toBe(404);
    }
    expect(upstream).not.toHaveBeenCalled();
  });

  it("backend fora do ar vira 503; content-type ausente vira JSON", async () => {
    upstream.mockRejectedValueOnce(new Error("timeout"));
    expect((await GET(new NextRequest("http://app/x"), params("v1", "branding"))).status).toBe(503);
    upstream.mockResolvedValueOnce(new Response("{}", { status: 200, headers: {} }));
    const res = await GET(new NextRequest("http://app/x"), params("v1", "branding"));
    expect(res.headers.get("content-type")).toMatch(/json|text/);
  });

  it("arquivos passam em bytes, com o nome; rota autenticada fica de fora", async () => {
    const bytes = new Uint8Array([0xef, 0xbb, 0xbf, 0x41]);
    upstream.mockResolvedValueOnce(
      new Response(bytes, {
        status: 200,
        headers: { "content-type": "text/csv; charset=utf-8", "content-disposition": 'attachment; filename="lista.csv"' },
      }),
    );
    const res = await GET(new NextRequest("http://app/x"), params("v1", "blog", "posts", "exportar"));
    expect(res.headers.get("content-disposition")).toBe('attachment; filename="lista.csv"');
    expect(new Uint8Array(await res.arrayBuffer())).toEqual(bytes);
    for (const path of [
      ["v1", "catalog", "services"],
      ["v1", "blog", "posts"],
    ]) {
      expect((await GET(new NextRequest("http://app/x"), params(...path))).status).toBe(200);
    }
    expect((await GET(new NextRequest("http://app/x"), params("v1", "me"))).status).toBe(404);
    expect((await POST(new NextRequest("http://app/x", { method: "POST", body: "{}" }), params("v1", "blog", "posts"))).status).toBe(404);
  });
});
