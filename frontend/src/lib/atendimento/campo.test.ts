import { afterEach, describe, expect, it, vi } from "vitest";

import { fail, mockBackend } from "@/test/backend";

import {
  ArmazemIndexedDB,
  ArmazemMemoria,
  FilaCampo,
  filaCampo,
  lerPosicao,
  novoId,
  usarFila,
  type Acao,
} from "./campo";

const ponto = { lat: -23.55, lng: -46.63, precisao_m: 8 };

function checkin(visitaId = "v1"): Acao {
  return {
    id: novoId(),
    tipo: "checkin",
    solicitacaoId: "s1",
    visitaId,
    local: "Escola",
    ponto,
    capturadaEm: "2026-10-21T12:00:00Z",
  };
}

function foto(objectKey?: string): Acao {
  return {
    id: novoId(),
    tipo: "evidencia",
    visitaId: "v1",
    evidencia: "foto",
    nome: "f.jpg",
    arquivo: new Blob(["jpeg"], { type: "image/jpeg" }),
    objectKey,
  };
}

const TICKET = {
  object_key: "atendimento/visitas/v1/x.jpg",
  upload_url: "https://minio.test/b/x?put",
  method: "PUT",
  headers: {},
  expires_at: "",
};

afterEach(() => usarFila(null));

describe("fila de campo", () => {
  it("envia em ordem: chegada, foto (upload direto + registro), saída e cancelamento", async () => {
    const api = mockBackend({
      "POST v1/atendimento/solicitacoes/s1/visitas": { status: 201, data: {} },
      "POST v1/atendimento/visitas/v1/evidencias/upload": { data: TICKET },
      "PUT minio.test/b/x": { status: 200 },
      "POST v1/atendimento/visitas/v1/evidencias": { status: 201, data: {} },
      "POST v1/atendimento/visitas/v1/checkout": { data: {} },
      "POST v1/atendimento/visitas/v2/cancelar": { data: {} },
    });
    const fila = new FilaCampo(new ArmazemMemoria());
    await fila.enfileirar(checkin());
    await fila.enfileirar(foto());
    await fila.enfileirar({
      id: novoId(),
      tipo: "checkout",
      visitaId: "v1",
      ponto,
      capturadaEm: "2026-10-21T13:00:00Z",
      assinanteNome: "Ana",
      observacao: "",
    });
    await fila.enfileirar({ id: novoId(), tipo: "cancelar", visitaId: "v2" });

    const r = await fila.processar();
    expect(r).toEqual({ enviadas: 4, pendentes: 0, recusadas: [] });
    expect(await fila.pendentes()).toHaveLength(0);
    expect(api.calls.map((c) => `${c.method} ${c.path}`)).toEqual([
      "POST v1/atendimento/solicitacoes/s1/visitas",
      "POST v1/atendimento/visitas/v1/evidencias/upload",
      "PUT minio.test/b/x",
      "POST v1/atendimento/visitas/v1/evidencias",
      "POST v1/atendimento/visitas/v1/checkout",
      "POST v1/atendimento/visitas/v2/cancelar",
    ]);
    expect(api.to("POST v1/atendimento/solicitacoes/s1/visitas")[0]!.body).toEqual({
      id: "v1",
      local: "Escola",
      destino: null,
      ponto,
      capturada_em: "2026-10-21T12:00:00Z",
    });
    expect(api.to("POST v1/atendimento/visitas/v1/evidencias")[0]!.body).toEqual({
      tipo: "foto",
      object_key: TICKET.object_key,
    });
    expect(api.to("POST v1/atendimento/visitas/v1/evidencias/upload")[0]!.body).toEqual({
      nome: "f.jpg",
      content_type: "image/jpeg",
    });
  });

  it("sem rede, para e guarda tudo; recusa definitiva é descartada", async () => {
    const fila = new FilaCampo(new ArmazemMemoria());
    await fila.enfileirar(checkin());
    await fila.enfileirar(checkin("v2"));
    vi.stubGlobal("fetch", vi.fn().mockRejectedValue(new TypeError("Failed to fetch")));
    expect(await fila.processar()).toEqual({ enviadas: 0, pendentes: 2, recusadas: [] });

    mockBackend({ "POST v1/atendimento/solicitacoes/s1/visitas": fail(503, "fora do ar") });
    expect((await fila.processar()).pendentes).toBe(2);
    mockBackend({ "POST v1/atendimento/solicitacoes/s1/visitas": fail(429, "devagar") });
    expect((await fila.processar()).pendentes).toBe(2);

    mockBackend({
      "POST v1/atendimento/solicitacoes/s1/visitas": fail(422, "localização inválida"),
    });
    const r = await fila.processar();
    expect(r.enviadas).toBe(0);
    expect(r.recusadas.map((x) => x.motivo)).toEqual([
      "localização inválida",
      "localização inválida",
    ]);
    expect(await fila.pendentes()).toHaveLength(0);
  });

  it("foto que já subiu não sobe de novo no reenvio", async () => {
    const armazem = new ArmazemMemoria();
    const fila = new FilaCampo(armazem);
    await fila.enfileirar(foto());
    const api = mockBackend({
      "POST v1/atendimento/visitas/v1/evidencias/upload": { data: TICKET },
      "PUT minio.test/b/x": { status: 200 },
      "POST v1/atendimento/visitas/v1/evidencias": fail(503, "banco fora"),
    });
    expect((await fila.processar()).pendentes).toBe(1);
    const [guardada] = await fila.pendentes();
    expect(guardada).toMatchObject({ objectKey: TICKET.object_key });

    const replay = mockBackend({
      "POST v1/atendimento/visitas/v1/evidencias": { status: 201, data: {} },
    });
    expect((await fila.processar()).enviadas).toBe(1);
    expect(replay.calls.map((c) => c.path)).toEqual(["v1/atendimento/visitas/v1/evidencias"]);
    expect(api.to("PUT minio.test/b/x")).toHaveLength(1);

    // Upload recusado pelo armazenamento: passageiro, fica.
    await fila.enfileirar(foto());
    mockBackend({
      "POST v1/atendimento/visitas/v1/evidencias/upload": { data: TICKET },
      "PUT minio.test/b/x": { status: 500 },
    });
    expect((await fila.processar()).pendentes).toBe(1);
    // Blob sem tipo vai como JPEG.
    const semTipo = {
      ...(foto() as Extract<Acao, { tipo: "evidencia" }>),
      arquivo: new Blob(["x"]),
    };
    const tipo = mockBackend({
      "POST v1/atendimento/visitas/v1/evidencias/upload": fail(422, "tipo"),
    });
    const outra = new FilaCampo(new ArmazemMemoria());
    await outra.enfileirar(semTipo);
    await outra.processar();
    expect(tipo.to("POST v1/atendimento/visitas/v1/evidencias/upload")[0]!.body).toEqual({
      nome: "f.jpg",
      content_type: "image/jpeg",
    });
  });

  it("uma rodada por vez", async () => {
    mockBackend({ "POST v1/atendimento/solicitacoes/s1/visitas": { status: 201, data: {} } });
    const fila = new FilaCampo(new ArmazemMemoria());
    await fila.enfileirar(checkin());
    const [a, b] = await Promise.all([fila.processar(), fila.processar()]);
    expect(a).toBe(b);
    expect(a.enviadas).toBe(1);
  });

  it("ids em ordem e a fila do aparelho", () => {
    const [x, y] = [novoId(), novoId()];
    expect(x < y).toBe(true);
    const f = filaCampo();
    expect(filaCampo()).toBe(f);
  });
});

describe("armazém no IndexedDB", () => {
  it("guarda, lista e remove", async () => {
    const armazem = new ArmazemIndexedDB(fakeIndexedDB());
    const a = checkin();
    await armazem.salvar(a);
    await armazem.salvar(checkin("v2"));
    expect((await armazem.todas()).map((x) => x.id)).toContain(a.id);
    await armazem.remover(a.id);
    expect(await armazem.todas()).toHaveLength(1);
  });

  it("falha ao abrir vira erro", async () => {
    const armazem = new ArmazemIndexedDB(fakeIndexedDB(true));
    await expect(armazem.todas()).rejects.toThrow("indisponível");
  });

  it("a fila do aparelho usa o IndexedDB quando existe", () => {
    vi.stubGlobal("indexedDB", fakeIndexedDB());
    usarFila(null);
    expect(filaCampo()).toBeInstanceOf(FilaCampo);
    vi.unstubAllGlobals();
  });
});

describe("leitura do GPS", () => {
  function geo(resposta: "ok" | number): Geolocation {
    return {
      getCurrentPosition: (ok: PositionCallback, erro?: PositionErrorCallback | null) => {
        if (resposta === "ok")
          ok({ coords: { latitude: 1, longitude: 2, accuracy: 7 } } as GeolocationPosition);
        else erro?.({ code: resposta } as GeolocationPositionError);
      },
    } as Geolocation;
  }

  it("lê, explica a recusa e a falha", async () => {
    await expect(lerPosicao(geo("ok"))).resolves.toEqual({ lat: 1, lng: 2, precisao_m: 7 });
    await expect(lerPosicao(geo(1))).rejects.toThrow("negada");
    await expect(lerPosicao(geo(3))).rejects.toThrow("Não foi possível ler");
    await expect(lerPosicao(undefined)).rejects.toThrow("não informa");
  });
});

/** IndexedDB mínimo em memória, só o que o armazém usa. */
function fakeIndexedDB(quebrado = false): IDBFactory {
  const dados = new Map<string, unknown>();
  const pedido = <T>(valor: () => T) => {
    const req = {} as IDBRequest<T> & { onsuccess?: () => void };
    setTimeout(() => {
      Object.assign(req, { result: valor() });
      req.onsuccess?.();
    }, 0);
    return req;
  };
  const loja = {
    getAll: () => pedido(() => [...dados.values()]),
    put: (v: { id: string }) => pedido(() => void dados.set(v.id, v)),
    delete: (id: string) => pedido(() => void dados.delete(id)),
  };
  const banco = {
    createObjectStore: () => loja,
    transaction: () => ({ objectStore: () => loja }),
  };
  return {
    open: () => {
      const req = {} as IDBOpenDBRequest & {
        onupgradeneeded?: () => void;
        onsuccess?: () => void;
        onerror?: () => void;
      };
      setTimeout(() => {
        if (quebrado) {
          Object.assign(req, { error: new Error("IndexedDB indisponível") });
          req.onerror?.();
          return;
        }
        Object.assign(req, { result: banco });
        req.onupgradeneeded?.();
        req.onsuccess?.();
      }, 0);
      return req;
    },
  } as unknown as IDBFactory;
}
