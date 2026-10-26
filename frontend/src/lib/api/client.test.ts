import { afterEach, describe, expect, it, vi } from "vitest";

import { apiClient, ApiError } from "./client";

function mockFetchOnce(status: number, body: unknown) {
  vi.stubGlobal(
    "fetch",
    vi.fn().mockResolvedValue({
      ok: status >= 200 && status < 300,
      status,
      json: async () => body,
    }),
  );
}

// jsdom não implementa navegação: troca window.location por um objeto
// simples o bastante pra location.replace(...) não lançar.
function fingirLocation(pathname: string) {
  const replace = vi.fn();
  Object.defineProperty(window, "location", {
    configurable: true,
    value: { pathname, search: "", origin: "http://localhost:3000", replace },
  });
  return replace;
}

describe("apiClient", () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("returns data on a successful envelope", async () => {
    mockFetchOnce(200, { data: { id: "1" }, error: null });
    const { data } = await apiClient.get<{ id: string }>("v1/integrations");
    expect(data).toEqual({ id: "1" });
  });

  it("sends a fresh X-Idempotency-Key on mutations, never on reads", async () => {
    mockFetchOnce(200, { data: {}, error: null });
    await apiClient.post("v1/blog/posts", { title: "x" });
    await apiClient.post("v1/blog/posts", { title: "x" });
    await apiClient.get("v1/blog/posts");
    const calls = (fetch as unknown as { mock: { calls: [string, RequestInit][] } }).mock.calls;
    const key = (i: number) =>
      (calls[i]![1].headers as Record<string, string>)["X-Idempotency-Key"];
    expect(key(0)).toMatch(/^[0-9a-f-]{36}$/);
    expect(key(1)).not.toBe(key(0));
    expect(key(2)).toBeUndefined();
  });

  it("passes through meta alongside data", async () => {
    mockFetchOnce(200, { data: [], error: null, meta: { page: 1 } });
    const { meta } = await apiClient.get<unknown[]>("v1/users");
    expect(meta).toEqual({ page: 1 });
  });

  it("throws ApiError when the envelope carries an error", async () => {
    mockFetchOnce(422, { data: null, error: { code: "VALIDATION_ERROR", message: "bad input" } });
    await expect(apiClient.post("v1/integrations/example/test")).rejects.toMatchObject({
      code: "VALIDATION_ERROR",
      message: "bad input",
      status: 422,
    });
  });

  it("throws ApiError when the response is ok but carries an error anyway", async () => {
    mockFetchOnce(200, { data: null, error: { code: "WEIRD", message: "should not happen" } });
    await expect(apiClient.get("v1/integrations")).rejects.toBeInstanceOf(ApiError);
  });

  it("204 é sucesso sem corpo (não lê JSON)", async () => {
    const json = vi.fn();
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue({ ok: true, status: 204, json }));
    const { data } = await apiClient.delete("v1/wiki/paginas/n1");
    expect(data).toBeUndefined();
    expect(json).not.toHaveBeenCalled();
  });

  it("throws a generic ApiError when the response body is not JSON", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue({
        ok: false,
        status: 500,
        json: async () => {
          throw new Error("not json");
        },
      }),
    );
    await expect(apiClient.get("v1/integrations")).rejects.toBeInstanceOf(ApiError);
  });

  // Fase 10 (projeto criado por upload .zip): postForm nunca pode fixar
  // Content-Type: application/json — isso sobrescreveria o
  // multipart/form-data; boundary=... que o browser precisa gerar
  // sozinho a partir do FormData, quebrando o parsing do lado do
  // servidor (o boundary some).
  it("postForm never sets a Content-Type header, letting the browser generate the multipart boundary", async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      status: 201,
      json: async () => ({ data: { id: "1" }, error: null }),
    });
    vi.stubGlobal("fetch", fetchMock);

    const form = new FormData();
    form.set("name", "test-project");
    await apiClient.postForm("v1/examples/upload", form);

    const [, init] = fetchMock.mock.calls[0] ?? [];
    const headers = init.headers as Record<string, string>;
    expect(headers["Content-Type"]).toBeUndefined();
    expect(init.body).toBe(form);
  });

  // Achado de auditoria: o usuário era redirecionado de volta ao login
  // sem nenhuma explicação de por quê — reason=session_expired é o que
  // permite ao LoginCard mostrar um aviso amigável em vez de deixar
  // parecer um bug aleatório.
  it("um 401 do proxy BFF redireciona a /login com reason=session_expired", async () => {
    mockFetchOnce(401, { data: null, error: { code: "UNAUTHORIZED", message: "sessão expirada" } });

    const replace = fingirLocation("/dashboard");

    await expect(apiClient.get("v1/integrations")).rejects.toBeInstanceOf(ApiError);

    const replaced = replace.mock.calls[0]?.[0] as string;
    expect(replaced).toContain("/login");
    expect(replaced).toContain("reason=session_expired");
    expect(replaced).toContain("callbackUrl=%2Fdashboard");
  });

  it("401 com código próprio (senha errada no Signum) NÃO derruba a sessão", async () => {
    mockFetchOnce(401, {
      data: null,
      error: {
        code: "SIGNUM_REAUTH_FAILED",
        message: "senha incorreta — a assinatura não foi realizada",
      },
    });
    const replace = fingirLocation("/signum/1");

    const err = await apiClient.post("v1/signum/envelopes/1/sign", {}).catch((e: unknown) => e);
    expect(err).toBeInstanceOf(ApiError);
    expect((err as ApiError).code).toBe("SIGNUM_REAUTH_FAILED");
    expect(replace).not.toHaveBeenCalled();
  });

  it("401 com corpo ilegível ainda é tratado como sessão expirada", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue({
        ok: false,
        status: 401,
        json: async () => {
          throw new Error("html");
        },
      }),
    );
    const replace = fingirLocation("/dashboard");

    await expect(apiClient.get("v1/me")).rejects.toMatchObject({ code: "INVALID_RESPONSE" });
    expect(replace).toHaveBeenCalled();
  });

  it("post() (JSON body) keeps setting Content-Type: application/json, unaffected by postForm's exception", async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      status: 200,
      json: async () => ({ data: null, error: null }),
    });
    vi.stubGlobal("fetch", fetchMock);

    await apiClient.post("v1/scanning/scans", { target: "https://example.com/repo.git" });

    const [, init] = fetchMock.mock.calls[0] ?? [];
    const headers = init.headers as Record<string, string>;
    expect(headers["Content-Type"]).toBe("application/json");
  });
});
