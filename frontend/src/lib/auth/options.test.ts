// @vitest-environment node
import type { Account, User } from "next-auth";
import type { JWT } from "next-auth/jwt";
import { afterEach, describe, expect, it, vi } from "vitest";

vi.mock("server-only", () => ({}));

type Options = typeof import("./options");

async function load(env: Record<string, string | undefined> = {}): Promise<Options> {
  vi.resetModules();
  for (const k of ["KEYCLOAK_ISSUER_URL", "KEYCLOAK_FRONTEND_CLIENT_ID", "KEYCLOAK_FRONTEND_CLIENT_SECRET"]) vi.stubEnv(k, env[k] ?? "");
  vi.stubEnv("BACKEND_INTERNAL_URL", "http://api:8080");
  vi.spyOn(console, "warn").mockImplementation(() => {});
  vi.spyOn(console, "error").mockImplementation(() => {});
  return import("./options");
}

const KC = { KEYCLOAK_ISSUER_URL: "https://sso/realms/n", KEYCLOAK_FRONTEND_CLIENT_ID: "nova-web", KEYCLOAK_FRONTEND_CLIENT_SECRET: "s" };

function jwtOf(payload: Record<string, unknown>): string {
  return `h.${Buffer.from(JSON.stringify(payload)).toString("base64url")}.s`;
}

// next-auth v4: o id próprio do provider fica em options.id
const idOf = (p: unknown) => (p as { options?: { id?: string }; id: string }).options?.id ?? (p as { id: string }).id;

function authorize(o: Options) {
  const local = o.authOptions.providers.find((p) => idOf(p) === "local") as unknown as {
    options: { authorize: (c: Record<string, string> | undefined, req?: { headers: Record<string, string> }) => Promise<unknown> };
  };
  return (c?: Record<string, string>, req?: { headers: Record<string, string> }) => local.options.authorize(c, req);
}

type JwtParams = { token: JWT; account?: Partial<Account> | null; user?: Partial<User>; trigger?: "signIn" | "update" };
const jwt = (o: Options, p: JwtParams) => (o.authOptions.callbacks!.jwt as unknown as (p: JwtParams) => Promise<JWT>)(p);

afterEach(() => {
  vi.unstubAllEnvs();
  vi.unstubAllGlobals();
  vi.restoreAllMocks();
});

describe("authOptions — provedores", () => {
  it("sem Keycloak só há o login local (com aviso)", async () => {
    const o = await load();
    expect(o.keycloakEnabled).toBe(false);
    expect(o.authOptions.providers.map(idOf)).toEqual(["local"]);
    expect(console.warn).toHaveBeenCalled();
    expect(o.authOptions.pages?.signIn).toBe("/login");
  });

  it("com Keycloak configurado, os dois", async () => {
    const o = await load(KC);
    expect(o.keycloakEnabled).toBe(true);
    expect(o.authOptions.providers.map(idOf)).toEqual(["keycloak", "local"]);
  });
});

describe("login local (authorize)", () => {
  it("credenciais vazias nem chamam o backend", async () => {
    const o = await load();
    const f = vi.fn();
    vi.stubGlobal("fetch", f);
    expect(await authorize(o)(undefined)).toBeNull();
    expect(await authorize(o)({ username: "a", password: "" })).toBeNull();
    expect(f).not.toHaveBeenCalled();
  });

  it("sucesso devolve o usuário com o token RS256 do backend", async () => {
    const o = await load();
    const f = vi.fn(async () => Response.json({ data: { access_token: "tok", expires_at: "2030-01-01T00:00:00Z", user: { id: "u1", username: "ana", email: "a@x" } }, error: null }));
    vi.stubGlobal("fetch", f);
    const u = await authorize(o)({ username: "ana", password: "Senha-123" });
    expect(u).toEqual({ id: "u1", name: "ana", email: "a@x", accessToken: "tok", accessTokenExpires: Date.parse("2030-01-01T00:00:00Z") });
    const [url, init] = f.mock.calls[0]! as unknown as [string, RequestInit];
    expect(url).toBe("http://api:8080/api/v1/auth/login");
    expect(JSON.parse(String(init.body))).toEqual({ username: "ana", password: "Senha-123" });
  });

  it("repassa o IP do visitante à API só atrás de proxy confiável", async () => {
    const o = await load();
    const f = vi.fn(async () => Response.json({ data: null, error: {} }, { status: 401 }));
    vi.stubGlobal("fetch", f);
    const req = { headers: { "x-forwarded-for": "203.0.113.7" } };
    await authorize(o)({ username: "ana", password: "x" }, req);
    expect((f.mock.calls[0]! as unknown as [string, RequestInit])[1].headers).not.toHaveProperty("X-Forwarded-For");
    vi.stubEnv("TRUST_PROXY_HEADERS", "true");
    await authorize(o)({ username: "ana", password: "x" }, req);
    expect((f.mock.calls[1]! as unknown as [string, RequestInit])[1].headers).toMatchObject({ "X-Forwarded-For": "203.0.113.7" });
  });

  it("nome de exibição, senha errada, corpo ilegível, backend fora e excesso de tentativas", async () => {
    const o = await load();
    vi.stubGlobal("fetch", vi.fn(async () => Response.json({ data: { access_token: "t", expires_at: "2030-01-01T00:00:00Z", user: { id: "u", username: "a", email: "", display_name: "Ana" } }, error: null })));
    expect(await authorize(o)({ username: "a", password: "b" })).toMatchObject({ name: "Ana" });

    vi.stubGlobal("fetch", vi.fn(async () => Response.json({ data: null, error: { code: "UNAUTHORIZED", message: "x" } }, { status: 401 })));
    expect(await authorize(o)({ username: "a", password: "b" })).toBeNull();

    vi.stubGlobal("fetch", vi.fn(async () => new Response("<html>", { status: 502 })));
    expect(await authorize(o)({ username: "a", password: "b" })).toBeNull();

    vi.stubGlobal("fetch", vi.fn(async () => { throw new Error("ECONNREFUSED"); }));
    expect(await authorize(o)({ username: "a", password: "b" })).toBeNull();

    // 429 do rate limiter, mesmo sem corpo JSON
    vi.stubGlobal("fetch", vi.fn(async () => new Response("Too Many Requests", { status: 429 })));
    await expect(authorize(o)({ username: "a", password: "b" })).rejects.toThrow("TooManyAttempts");
  });
});

describe("callback jwt", () => {
  const future = () => Date.now() + 3600_000;

  it("login Keycloak guarda os tokens e as claims de exibição (roles de realm e do client, grupos)", async () => {
    const o = await load(KC);
    const access = jwtOf({ roles: ["a"], realm_access: { roles: ["b"] }, resource_access: { "nova-web": { roles: ["c"] } }, groups: ["GRP_TI"] });
    const t = await jwt(o, { token: {}, account: { provider: "keycloak", access_token: access, expires_at: Math.floor(future() / 1000), refresh_token: "r", id_token: "i" } });
    expect(t).toMatchObject({ accessToken: access, refreshToken: "r", idToken: "i", provider: "keycloak", roles: ["a", "b", "c"], groups: ["GRP_TI"] });
    expect(t.error).toBeUndefined();
  });

  it("login local limpa refresh/id token; token ilegível não quebra", async () => {
    const o = await load();
    const t = await jwt(o, { token: { refreshToken: "velho" }, account: { provider: "local" }, user: { accessToken: "sem-payload", accessTokenExpires: future() } as Partial<User> });
    expect(t).toMatchObject({ provider: "local", roles: [], groups: [] });
    expect(t.refreshToken).toBeUndefined();
    const bad = await jwt(o, { token: { accessToken: "h.%%%.s", accessTokenExpires: future() } });
    expect(bad).toMatchObject({ roles: [], groups: [] });
  });

  it("token local vencido marca erro (sem refresh)", async () => {
    const o = await load();
    const t = await jwt(o, { token: { accessToken: jwtOf({}), accessTokenExpires: Date.now() - 1000 } });
    expect(t.error).toBe("RefreshAccessTokenError");
  });

  it("Keycloak vencido é renovado com o refresh_token", async () => {
    const o = await load(KC);
    const f = vi.fn(async () => Response.json({ access_token: jwtOf({ groups: ["G"] }), expires_in: 300 }));
    vi.stubGlobal("fetch", f);
    const t = await jwt(o, { token: { accessToken: jwtOf({}), accessTokenExpires: Date.now() - 1000, refreshToken: "r1", idToken: "i1" } });
    expect(t.error).toBeUndefined();
    expect(t.refreshToken).toBe("r1");
    expect(t.idToken).toBe("i1");
    expect(t.accessTokenExpires).toBeGreaterThan(Date.now());
    const [url, init] = f.mock.calls[0]! as unknown as [string, RequestInit];
    expect(url).toBe("https://sso/realms/n/protocol/openid-connect/token");
    expect(String(init.body)).toContain("grant_type=refresh_token");

    vi.stubGlobal("fetch", vi.fn(async () => Response.json({ access_token: "novo", expires_in: 60, refresh_token: "r2", id_token: "i2" })));
    const t2 = await jwt(o, { token: { accessToken: "x", accessTokenExpires: 1, refreshToken: "r1" } });
    expect(t2).toMatchObject({ accessToken: "novo", refreshToken: "r2", idToken: "i2" });
  });

  it("refresh recusado, sem access_token ou com expires_in inválido marca erro", async () => {
    const o = await load(KC);
    for (const body of [{ error: "invalid_grant" }, { expires_in: 60 }, { access_token: "a", expires_in: 0 }, { access_token: "a", expires_in: "x" }]) {
      vi.stubGlobal("fetch", vi.fn(async () => Response.json(body, { status: body.error ? 400 : 200 })));
      const t = await jwt(o, { token: { accessToken: "x", accessTokenExpires: 1, refreshToken: "r" } });
      expect(t.error).toBe("RefreshAccessTokenError");
    }
    vi.stubGlobal("fetch", vi.fn(async () => { throw new Error("rede"); }));
    expect((await jwt(o, { token: { accessToken: "x", accessTokenExpires: 1, refreshToken: "r" } })).error).toBe("RefreshAccessTokenError");
    // o erro bruto (que pode ecoar o refresh_token) nunca é logado
    expect(vi.mocked(console.error).mock.calls.every((c) => c.length === 1)).toBe(true);
  });

  it("update() renova o token local perto do vencimento (sessão por inatividade)", async () => {
    const o = await load();
    const quase = Date.now() + 5 * 60_000;
    const f = vi.fn(async () => Response.json({ data: { access_token: jwtOf({ roles: ["nova"] }), expires_at: "2030-01-01T00:00:00Z", user: { id: "u", username: "a", email: "" } }, error: null }));
    vi.stubGlobal("fetch", f);

    // Leitura comum da sessão (sem trigger) nunca renova.
    expect((await jwt(o, { token: { provider: "local", accessToken: "velho", accessTokenExpires: quase } })).accessToken).toBe("velho");
    // Longe do vencimento, nem o update() renova.
    expect((await jwt(o, { token: { provider: "local", accessToken: "velho", accessTokenExpires: future() }, trigger: "update" })).accessToken).toBe("velho");
    expect(f).not.toHaveBeenCalled();

    const t = await jwt(o, { token: { provider: "local", accessToken: "velho", accessTokenExpires: quase }, trigger: "update" });
    expect(t).toMatchObject({ accessTokenExpires: Date.parse("2030-01-01T00:00:00Z"), roles: ["nova"] });
    expect(t.error).toBeUndefined();
    const [url, init] = f.mock.calls[0]! as unknown as [string, RequestInit];
    expect(url).toBe("http://api:8080/api/v1/auth/refresh");
    expect(init.headers).toMatchObject({ Authorization: "Bearer velho" });
  });

  it("renovação local recusada ou com falha mantém o token atual até vencer", async () => {
    const o = await load();
    const token = { provider: "local", accessToken: "velho", accessTokenExpires: Date.now() + 60_000 };
    for (const resp of [
      () => Response.json({ data: null, error: { code: "FORBIDDEN", message: "teto" } }, { status: 403 }),
      () => Response.json({ data: { access_token: "x", expires_at: "lixo", user: {} }, error: null }),
      () => { throw new Error("rede"); },
    ]) {
      vi.stubGlobal("fetch", vi.fn(async () => resp()));
      const t = await jwt(o, { token: { ...token }, trigger: "update" });
      expect(t).toMatchObject({ accessToken: "velho" });
      expect(t.error).toBeUndefined();
    }
  });

  it("update() renova o Keycloak antes de vencer", async () => {
    const o = await load(KC);
    vi.stubGlobal("fetch", vi.fn(async () => Response.json({ access_token: "novo", expires_in: 900 })));
    const t = await jwt(o, { token: { provider: "keycloak", accessToken: "x", accessTokenExpires: Date.now() + 60_000, refreshToken: "r" }, trigger: "update" });
    expect(t).toMatchObject({ accessToken: "novo" });
    expect(t.accessTokenExpires).toBeGreaterThan(Date.now() + 10 * 60_000);
  });

  it("login Keycloak sem expires_at não tem vencimento conhecido", async () => {
    const o = await load(KC);
    const t = await jwt(o, { token: {}, account: { provider: "keycloak", access_token: jwtOf({}) } });
    expect(t.accessTokenExpires).toBeUndefined();
    expect(t.error).toBe("RefreshAccessTokenError");
  });
});

describe("callback session", () => {
  it("expõe só claims de exibição, provedor e erro", async () => {
    const o = await load();
    const session = (o.authOptions.callbacks!.session as unknown as (p: { session: Record<string, unknown>; token: JWT }) => Promise<Record<string, unknown>>);
    const s = await session({ session: { expires: "" }, token: { roles: ["r"], groups: ["g"], provider: "local", error: "E", accessToken: "segredo" } });
    expect(s).toEqual({ expires: "", user: { roles: ["r"], groups: ["g"] }, provider: "local", error: "E" });
    const s2 = await session({ session: { expires: "", user: { name: "A" } }, token: {} });
    expect(s2.user).toEqual({ name: "A", roles: [], groups: [] });
    // O vencimento (não o token) vai ao navegador para o SessionKeepAlive.
    expect((await session({ session: { expires: "" }, token: { accessToken: "segredo", accessTokenExpires: 123 } })).expiresAt).toBe(123);
  });
});
