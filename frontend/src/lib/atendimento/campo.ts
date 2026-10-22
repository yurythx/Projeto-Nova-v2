// Fila de campo: o técnico registra a visita mesmo sem rede. Cada ação
// fica guardada no aparelho e sobe quando a conexão volta. O reenvio é
// seguro porque o servidor reconhece a repetição: o check-in pelo id da
// visita (gerado aqui), o check-out pelo horário capturado e a evidência
// pela chave do objeto já enviado.

import { ApiError, apiClient } from "@/lib/api/client";
import type { Destino, Ponto } from "@/lib/atendimento/atendimento";
import type { UploadTicket } from "@/lib/sismarq/types";
import { putToTicket } from "@/lib/sismarq/upload";

export type Acao =
  | {
      id: string;
      tipo: "checkin";
      solicitacaoId: string;
      visitaId: string;
      local: string;
      destino?: Destino;
      ponto: Ponto;
      capturadaEm: string;
    }
  | {
      id: string;
      tipo: "checkout";
      visitaId: string;
      ponto: Ponto;
      capturadaEm: string;
      assinanteNome: string;
      observacao: string;
    }
  | { id: string; tipo: "cancelar"; visitaId: string }
  | {
      id: string;
      tipo: "evidencia";
      visitaId: string;
      evidencia: "foto" | "assinatura";
      nome: string;
      arquivo: Blob;
      /** Preenchido quando o arquivo já subiu: o reenvio só registra. */
      objectKey?: string;
    };

/** Onde a fila guarda as ações. */
export interface Armazem {
  todas(): Promise<Acao[]>;
  salvar(acao: Acao): Promise<void>;
  remover(id: string): Promise<void>;
}

/** Armazém em memória: testes e navegadores sem IndexedDB (a fila vale
 * enquanto a página estiver aberta). */
export class ArmazemMemoria implements Armazem {
  private acoes = new Map<string, Acao>();
  async todas() {
    return [...this.acoes.values()];
  }
  async salvar(acao: Acao) {
    this.acoes.set(acao.id, acao);
  }
  async remover(id: string) {
    this.acoes.delete(id);
  }
}

const BANCO = "atendimento-campo";
const LOJA = "acoes";

function pedido<T>(req: IDBRequest<T>): Promise<T> {
  return new Promise((ok, erro) => {
    req.onsuccess = () => ok(req.result);
    req.onerror = () => erro(req.error);
  });
}

/** Armazém no IndexedDB do aparelho: sobrevive a fechar a página. A ordem
 * de inserção é preservada pela chave (id ordenável por tempo). */
export class ArmazemIndexedDB implements Armazem {
  private banco: Promise<IDBDatabase>;
  constructor(idb: IDBFactory) {
    const abrir = idb.open(BANCO, 1);
    abrir.onupgradeneeded = () => abrir.result.createObjectStore(LOJA, { keyPath: "id" });
    this.banco = pedido(abrir);
  }
  private async loja(modo: IDBTransactionMode) {
    return (await this.banco).transaction(LOJA, modo).objectStore(LOJA);
  }
  async todas() {
    return pedido((await this.loja("readonly")).getAll() as IDBRequest<Acao[]>);
  }
  async salvar(acao: Acao) {
    await pedido((await this.loja("readwrite")).put(acao));
  }
  async remover(id: string) {
    await pedido((await this.loja("readwrite")).delete(id));
  }
}

/** Recusa definitiva do servidor: não adianta reenviar. Falha de rede,
 * 5xx, 408 e 429 são passageiras. */
function definitiva(err: unknown): boolean {
  return (
    err instanceof ApiError &&
    err.status >= 400 &&
    err.status < 500 &&
    err.status !== 408 &&
    err.status !== 429
  );
}

export interface Resultado {
  enviadas: number;
  pendentes: number;
  recusadas: { acao: Acao; motivo: string }[];
}

let sequencia = 0;

/** Id ordenável: o instante e um contador garantem a ordem de envio. */
export function novoId(): string {
  sequencia += 1;
  return `${Date.now().toString().padStart(15, "0")}-${String(sequencia).padStart(6, "0")}-${crypto.randomUUID()}`;
}

export class FilaCampo {
  private rodando: Promise<Resultado> | null = null;
  constructor(private armazem: Armazem) {}

  async enfileirar(acao: Acao): Promise<void> {
    await this.armazem.salvar(acao);
  }

  async pendentes(): Promise<Acao[]> {
    return (await this.armazem.todas()).sort((a, b) => a.id.localeCompare(b.id));
  }

  /** Envia em ordem. Para na primeira falha passageira (o resto depende
   * dela, ex.: o check-out depois do check-in); descarta a recusada. Uma
   * chamada por vez. */
  processar(): Promise<Resultado> {
    this.rodando ??= this.rodar().finally(() => {
      this.rodando = null;
    });
    return this.rodando;
  }

  private async rodar(): Promise<Resultado> {
    const r: Resultado = { enviadas: 0, pendentes: 0, recusadas: [] };
    const fila = await this.pendentes();
    for (let i = 0; i < fila.length; i++) {
      const acao = fila[i]!;
      try {
        await this.enviar(acao);
        await this.armazem.remover(acao.id);
        r.enviadas++;
      } catch (err) {
        if (!definitiva(err)) {
          r.pendentes = fila.length - i;
          break;
        }
        await this.armazem.remover(acao.id);
        r.recusadas.push({ acao, motivo: (err as ApiError).message });
      }
    }
    return r;
  }

  private async enviar(acao: Acao): Promise<void> {
    switch (acao.tipo) {
      case "checkin":
        await apiClient.post(`v1/atendimento/solicitacoes/${acao.solicitacaoId}/visitas`, {
          id: acao.visitaId,
          local: acao.local,
          destino: acao.destino ?? null,
          ponto: acao.ponto,
          capturada_em: acao.capturadaEm,
        });
        return;
      case "checkout":
        await apiClient.post(`v1/atendimento/visitas/${acao.visitaId}/checkout`, {
          ponto: acao.ponto,
          capturada_em: acao.capturadaEm,
          assinante_nome: acao.assinanteNome,
          observacao: acao.observacao,
        });
        return;
      case "cancelar":
        await apiClient.post(`v1/atendimento/visitas/${acao.visitaId}/cancelar`);
        return;
      case "evidencia":
        if (!acao.objectKey) {
          const tipoMime = acao.arquivo.type || "image/jpeg";
          const { data: ticket } = await apiClient.post<UploadTicket>(
            `v1/atendimento/visitas/${acao.visitaId}/evidencias/upload`,
            { nome: acao.nome, content_type: tipoMime },
          );
          await putToTicket(ticket, acao.arquivo);
          // Já subiu: se o registro falhar, o reenvio não manda de novo.
          acao.objectKey = ticket.object_key;
          await this.armazem.salvar(acao);
        }
        await apiClient.post(`v1/atendimento/visitas/${acao.visitaId}/evidencias`, {
          tipo: acao.evidencia,
          object_key: acao.objectKey,
        });
        return;
    }
  }
}

let unica: FilaCampo | null = null;

/** A fila do aparelho (IndexedDB quando houver). */
export function filaCampo(): FilaCampo {
  unica ??= new FilaCampo(
    typeof indexedDB === "undefined" ? new ArmazemMemoria() : new ArmazemIndexedDB(indexedDB),
  );
  return unica;
}

/** Troca a fila (testes). */
export function usarFila(fila: FilaCampo | null): void {
  unica = fila;
}

/** Lê o GPS uma vez, com alta precisão. Recusa ou demora viram erro com
 * mensagem para a pessoa. */
export function lerPosicao(
  geo: Geolocation | undefined = globalThis.navigator?.geolocation,
): Promise<Ponto> {
  return new Promise((ok, erro) => {
    if (!geo) {
      erro(new Error("Este aparelho não informa a localização."));
      return;
    }
    geo.getCurrentPosition(
      (p) => ok({ lat: p.coords.latitude, lng: p.coords.longitude, precisao_m: p.coords.accuracy }),
      (e) =>
        erro(
          new Error(
            e.code === 1
              ? "A localização foi negada. Permita o acesso à localização para registrar a visita."
              : "Não foi possível ler a localização. Tente de novo em local aberto.",
          ),
        ),
      { enableHighAccuracy: true, timeout: 20_000, maximumAge: 0 },
    );
  });
}
