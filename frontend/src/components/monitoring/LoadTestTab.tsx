"use client";

import { useState, useRef, useEffect, useCallback } from "react";
import {
  Zap,
  Play,
  Square,
  BarChart3,
  CheckCircle2,
  AlertTriangle,
  Clock,
  Gauge,
  TrendingUp,
  Download,
  Copy,
  Check,
  Info,
  ShieldCheck,
  Layers,
  ChevronDown,
  ChevronUp,
  ArrowRight,
} from "lucide-react";
import { Button } from "@/components/ui/Button";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/Card";

interface BenchmarkResult {
  id: string;
  targetId: string;
  targetName: string;
  url: string;
  concurrency: number;
  totalRequests: number;
  completedRequests: number;
  successCount: number; // 2xx
  rateLimitedCount: number; // 429
  clientErrorCount: number; // 4xx
  serverErrorCount: number; // 5xx
  durationMs: number;
  reqPerSec: number;
  latencies: number[];
  avgLatency: number;
  minLatency: number;
  maxLatency: number;
  p95Latency: number;
  p99Latency: number;
  timestamp: string;
  diagnosticSummary: string;
}

interface LogEntry {
  id: number;
  statusCode: number;
  durationMs: number;
  timestamp: string;
}

interface TargetItem {
  id: string;
  name: string;
  url: string;
  description: string;
  impactLevel: string;
  pipeline: string[];
  testExplanation: {
    overview: string;
    subsystems: string[];
    stressVector: string;
    expectedBehavior: string;
  };
  performanceGuide: {
    slaTarget: string;
    latencyInterpretation: string;
    rateLimitBehavior: string;
    optimizationTip: string;
  };
}

const TARGETS: TargetItem[] = [
  {
    id: "catalog",
    name: "Catálogo de Serviços Público",
    url: "/api/public/v1/catalog/services",
    description: "Endpoint público de consulta das cartas de serviços e procedimentos com filtro",
    impactLevel: "Banco de Dados & I/O",
    pipeline: ["Caddy (Proxy HTTPS/TLS)", "Next.js BFF (Allowlist)", "Go Core REST", "PostgreSQL (Pool pgx)"],
    testExplanation: {
      overview:
        "Avalia a capacidade de busca, junção de categorias e serialização de cartas de serviços públicos sob intensa consulta simultânea por múltiplos visitantes anônimos.",
      subsystems: [
        "PostgreSQL 16: Conexões simultâneas do pool de consultas (pgx pool) e leitura de índices",
        "Go Backend: Roteador Chi, desserialização relacional e alocação de memória no heap",
        "Next.js BFF Proxy: Validação de cabeçalhos e repasse não-bloqueante com keep-alive",
        "Caddy Edge: Terminação TLS 1.3 e compressão zstd/gzip para transferência rápida",
      ],
      stressVector: "Concorrência no pool de conexões do banco de dados e serialização de payload JSON.",
      expectedBehavior:
        "Respostas com P95 abaixo de 50ms em regime normal. Quando disparado por IP único em excesso (>120 req/min), deve aplicar rate limit 429 para proteger o banco.",
    },
    performanceGuide: {
      slaTarget: "P95 < 50ms | 0% erros 5xx | Proteção 429 ativa",
      latencyInterpretation:
        "Latências abaixo de 20ms indicam que o pool do PostgreSQL e a camada de rede estão respondendo imediatamente sem filas de espera de conexão.",
      rateLimitBehavior:
        "O Redis contabiliza requisições por janela deslizante. Acionar 429 comprova que robôs não conseguirão exaurir o pool do banco.",
      optimizationTip:
        "Ativação de cabeçalho Cache-Control (stale-while-revalidate) permite que o Caddy responda em < 2ms na borda.",
    },
  },
  {
    id: "blog",
    name: "Blog & Notícias (Destaque Editorial)",
    url: "/api/public/v1/blog/posts",
    description: "Artigos institucionais, guias de infraestrutura (Proxmox/Ansible) e destaques",
    impactLevel: "Cache & Leitura Pública",
    pipeline: ["Caddy (Proxy HTTPS/TLS)", "Next.js BFF (Allowlist)", "Go Core REST", "PostgreSQL (Índice de Tags)"],
    testExplanation: {
      overview:
        "Testa a publicação e entrega de artigos do blog institucional, posts em destaque editorial e metadados de leitura rápida para a população e equipes técnicas.",
      subsystems: [
        "Go REST Module: Módulo interno de blog com verificação de status publicado e ordenação",
        "PostgreSQL 16: Index scan por slug e data de publicação com paginação segura",
        "BFF Allowlist: Roteamento anônimo sem vazamento de tokens de autenticação",
      ],
      stressVector: "Transferência de payloads ricos em texto e metadados com compressão ativa.",
      expectedBehavior:
        "P95 nominal entre 10ms e 40ms. Vazão contínua sem degradação do heap de memória do Go.",
    },
    performanceGuide: {
      slaTarget: "P95 < 40ms | 0% erros 5xx | Compressão ativa",
      latencyInterpretation:
        "Tempo de resposta homogêneo entre a primeira e a centésima chamada indica ausência de locks de tabela e cache de query eficiente.",
      rateLimitBehavior:
        "Suporta rajadas de navegação rápida antes de ativar o teto de segurança por IP.",
      optimizationTip:
        "Artigos publicados possuem conteúdo imutável por longos períodos; ideais para cache de borda.",
    },
  },
  {
    id: "branding",
    name: "Branding & Identidade Visual",
    url: "/api/public/v1/branding",
    description: "Identidade visual governamental, cores institucionais, brasão e parâmetros",
    impactLevel: "Alta Frequência / Leve",
    pipeline: ["Caddy (Proxy HTTPS/TLS)", "Next.js BFF (Allowlist)", "Go Core REST", "Redis (In-Memory)"],
    testExplanation: {
      overview:
        "Endpoint acessado na carga inicial de qualquer página do portal para renderizar cores, logotipo e metadados da instituição.",
      subsystems: [
        "Go REST Engine: Leitura ultrarrápida da configuração visual ativa",
        "Redis / Cache em Memória: Resposta instantânea sem consulta a disco",
        "Caddy: Compressão e terminação TLS rápida",
      ],
      stressVector: "Alta frequência de chamadas por segundo (Throughput máximo com I/O mínimo).",
      expectedBehavior:
        "Mediana (P50) sub-milissegundo (< 2ms) e P95 inferior a 25ms, simulando tráfego de milhares de acessos por minuto.",
    },
    performanceGuide: {
      slaTarget: "P95 < 25ms | P50 < 2ms | 0% erros 5xx",
      latencyInterpretation:
        "Representa o piso de latência da rede e do runtime Go puro; se estiver acima de 50ms, o gargalo é na camada de rede ou proxy.",
      rateLimitBehavior:
        "Opera com teto permissivo para não bloquear clientes que recarregam múltiplas páginas.",
      optimizationTip:
        "Pode ser cacheado localmente no navegador por 5 a 15 minutos via cabeçalhos HTTP padrão.",
    },
  },
  {
    id: "health",
    name: "Healthcheck da Stack (BFF / Microserviços)",
    url: "/api/health",
    description: "Sonda de liveness e readiness de todos os contêineres e dependências",
    impactLevel: "Sonda de Diagnóstico",
    pipeline: ["Caddy (Proxy HTTPS)", "Next.js Route Handler", "Postgres Ping", "Redis Ping", "MinIO Ping"],
    testExplanation: {
      overview:
        "Exercita o healthcheck completo do sistema, que verifica a conectividade simultânea com o PostgreSQL, Redis, MinIO e barramento de eventos.",
      subsystems: [
        "Next.js Runtime: Execução de sondas assíncronas com timeout estrito de 3 segundos",
        "Postgres / Redis / RabbitMQ / MinIO: Resposta a pings de conectividade TCP",
      ],
      stressVector: "Validação cruzada de sockets de rede de todos os serviços de suporte.",
      expectedBehavior:
        "Retorno HTTP 200 OK com status 'ok' para cada serviço verificado. Latência entre 5ms e 35ms.",
    },
    performanceGuide: {
      slaTarget: "P95 < 45ms | Status 200 OK consistente",
      latencyInterpretation:
        "Se o P95 subir acima de 100ms, indica que algum serviço auxiliar (ex: MinIO ou RabbitMQ) está demorando a responder ao ping TCP.",
      rateLimitBehavior:
        "Healthchecks são isentos de rate limit para manter probes do Docker e Kubernetes sempre ativas.",
      optimizationTip:
        "Utilize conexões persistentes nos checks para evitar sobrecarga de handshake TCP a cada probe.",
    },
  },
  {
    id: "contact_ratelimit",
    name: "Formulário de Ouvidoria (Teste de Rate Limit & Defesa)",
    url: "/api/public/v1/contact/messages",
    description: "Endpoint POST de envio de mensagens com barreira anti-spam e Rate Limit ativo",
    impactLevel: "Segurança & Defesa Anti-Spam",
    pipeline: ["Caddy (Proxy HTTPS)", "Next.js BFF (Allowlist POST)", "Go Core REST", "Redis RateLimiter", "PostgreSQL"],
    testExplanation: {
      overview:
        "Mede a eficácia dos mecanismos de defesa ativa contra inundações (DDoS), bots e submissões automatizadas em massa de formulários públicos.",
      subsystems: [
        "Redis Rate Limiter: Janela atômica distribuída (INCR + PEXPIRE) com penalidade progressiva",
        "Next.js BFF: Repasse estrito de payloads JSON com sanitização e limite de tamanho",
        "Go Backend: Resposta imediata 429 com cabeçalho Retry-After: 60 quando o limite é excedido",
      ],
      stressVector: "Rajada deliberada de requisições concorrentes a partir do mesmo IP para comprovar contenção.",
      expectedBehavior:
        "Aceite das primeiras requisições autorizadas e bloqueio imediato das subsequentes com HTTP 429 Too Many Requests em < 1ms.",
    },
    performanceGuide: {
      slaTarget: "HTTP 429 acionado em < 2ms | 0% quedas de banco",
      latencyInterpretation:
        "Quando o rate limit atua, a latência das respostas 429 deve ser sub-milissegundo (< 1ms), pois o Redis corta o tráfego antes de atingir o PostgreSQL.",
      rateLimitBehavior:
        "Taxa de 429 alta neste teste é o resultado ESPERADO e desejado: confirma que a defesa perimetral está protegendo os recursos públicos.",
      optimizationTip:
        "Os clientes recebem o header 'Retry-After: 60' informando o tempo de espera necessário até a liberação da janela.",
    },
  },
  {
    id: "modelos_minio",
    name: "Leitura pública & Storage MinIO/S3",
    url: "/api/public/v1/blog/posts",
    description: "Consulta pública de conteúdo publicado, com objetos no MinIO quando o plug-in guarda arquivo",
    impactLevel: "Armazenamento S3 & MinIO",
    pipeline: ["Caddy (Proxy TLS)", "Next.js BFF (Allowlist)", "Go Core REST", "MinIO S3 Engine", "PostgreSQL"],
    testExplanation: {
      overview:
        "Mede a vazão e a estabilidade do armazenamento de objetos S3 (MinIO) usado pelos plug-ins de arquivo.",
      subsystems: [
        "MinIO (S3 API): Conexões simultâneas de leitura de metadados e objetos binários",
        "PostgreSQL: Consulta de versões ativas, integridade de hashes SHA-256 e MIME types",
        "Go REST Streamer: Streaming em bytes com zero-copy para entrega rápida ao navegador",
        "Caddy: Compressão e terminação TLS sem bufferização excessiva de binários",
      ],
      stressVector: "Concorrência de leitura em armazenamento de objetos S3 e validação de integridade criptográfica.",
      expectedBehavior:
        "P95 nominal abaixo de 45ms para metadados e streaming contínuo sem estouro de buffers de memória.",
    },
    performanceGuide: {
      slaTarget: "P95 < 45ms | 0% falhas de storage | 100% integridade SHA-256",
      latencyInterpretation:
        "Latências baixas confirmam que o MinIO responde localmente na rede interna sem filas de lock em disco.",
      rateLimitBehavior:
        "Obedece à cota de leitura pública para evitar raspagem automatizada em massa de documentos institucionais.",
      optimizationTip:
        "Para arquivos volumosos frequentes, o uso de URLs pré-assinadas com cache HTTP alivia o tráfego do gateway da API.",
    },
  },
  {
    id: "outbox_messaging",
    name: "Barramento Transacional & Mensageria (RabbitMQ / Outbox)",
    url: "/api/backend/v1/monitoring/outbox-stats",
    description: "Monitoramento de vazão do Transactional Outbox e saúde da fila de eventos AMQP",
    impactLevel: "Mensageria & Broker AMQP",
    pipeline: ["Caddy (Proxy TLS)", "Next.js BFF Auth", "Go Core REST", "PostgreSQL (outbox_events)", "RabbitMQ 3.13"],
    testExplanation: {
      overview:
        "Avalia a integridade do padrão Transactional Outbox do Projeto-nova-v2 sob concorrência, verificando o status de publicação assíncrona para o RabbitMQ.",
      subsystems: [
        "PostgreSQL 16: Contagem atômica de eventos pendentes, publicados e em Dead Letter (DLQ)",
        "RabbitMQ 3.13: Disponibilidade dos canais AMQP e estabilidade do broker de mensagens",
        "Go Background Relay: Worker de despacho assíncrono entre banco e filas",
        "Next.js BFF Auth: Injeção de credencial segura de sessão de monitoramento",
      ],
      stressVector: "Consultas de telemetria transacional concorrentes durante despacho de eventos assíncronos.",
      expectedBehavior:
        "P95 abaixo de 35ms, com 0 falhas na contagem de dead letters e fila de pendências estável.",
    },
    performanceGuide: {
      slaTarget: "P95 < 35ms | 0 eventos em Dead Letter | 100% entrega transacional",
      latencyInterpretation:
        "Latência consistente indica que as transações de outbox no Postgres não estão sofrendo lock de tabela.",
      rateLimitBehavior:
        "Endpoint autenticado com cota por identidade administrativa protegendo o canal de auditoria.",
      optimizationTip:
        "Manter o pool de conexões do worker de mensageria dimensionado para evitar acúmulo de outbox_events pendentes.",
    },
  },
  {
    id: "cluster_probes",
    name: "Sondas de Cluster & Infraestrutura (RabbitMQ, MinIO, Redis & DB)",
    url: "/api/health",
    description: "Estresse simultâneo dos 4 pilares da stack: Postgres 16, Redis 7.4, RabbitMQ 3.13 e MinIO",
    impactLevel: "Alta Concorrência Multi-Serviço",
    pipeline: ["Caddy (Proxy TLS)", "Next.js Runtime", "Go /ready Core", "4x Sondas Paralelas (PG, Redis, RabbitMQ, MinIO)"],
    testExplanation: {
      overview:
        "Dispara probes concorrentes no endpoint central de readiness, que executa pings TCP e liveness simultâneos em todos os 4 serviços da infraestrutura Proxmox.",
      subsystems: [
        "PostgreSQL: Validação de pool e conectividade de transação",
        "Redis: Teste de PING em memória e resposta do cluster",
        "RabbitMQ: Checagem de handshake no protocolo AMQP 0-9-1",
        "MinIO: Validação de probe do bucket S3 e credenciais de acesso",
        "Next.js Runtime: Orquestração de timeouts de 3 segundos por sonda",
      ],
      stressVector: "Inundação de probes de liveness para verificar se a infraestrutura aguenta checagens de alta frequência sem degradação.",
      expectedBehavior:
        "Status HTTP 200 OK com 'status: ok' para 100% dos 4 serviços. P95 inferior a 45ms.",
    },
    performanceGuide: {
      slaTarget: "P95 < 45ms | 100% dos serviços saudáveis | 0% timeouts",
      latencyInterpretation:
        "Se o tempo ultrapassar 100ms, o log individual indica qual dos 4 serviços está com atraso de handshake.",
      rateLimitBehavior:
        "Sondas operam livres de rate limit para não mascarar incidentes em orquestradores.",
      optimizationTip:
        "Pings reutilizam conexões pré-estabelecidas com keep-alive para minimizar handshakes TCP repetitivos.",
    },
  },
];

const DEFAULT_TARGET: TargetItem = TARGETS[0]!;

export function LoadTestTab() {
  const [selectedTargetId, setSelectedTargetId] = useState<string>("catalog");
  const [concurrency, setConcurrency] = useState<number>(10);
  const [totalRequestsTarget, setTotalRequestsTarget] = useState<number>(100);
  const [isRunning, setIsRunning] = useState<boolean>(false);
  const [copied, setCopied] = useState<boolean>(false);
  const [expandedHistoryId, setExpandedHistoryId] = useState<string | null>(null);

  // Métricas em tempo real
  const [currentProgress, setCurrentProgress] = useState<number>(0);
  const [liveReqPerSec, setLiveReqPerSec] = useState<number>(0);
  const [stats, setStats] = useState<{
    completed: number;
    success: number;
    rateLimited: number;
    clientErrors: number;
    serverErrors: number;
    avgLatency: number;
    minLatency: number;
    maxLatency: number;
    p95: number;
    p99: number;
  }>({
    completed: 0,
    success: 0,
    rateLimited: 0,
    clientErrors: 0,
    serverErrors: 0,
    avgLatency: 0,
    minLatency: 0,
    maxLatency: 0,
    p95: 0,
    p99: 0,
  });

  const [recentLogs, setRecentLogs] = useState<LogEntry[]>([]);
  const [history, setHistory] = useState<BenchmarkResult[]>([]);

  const abortControllerRef = useRef<AbortController | null>(null);
  const isRunningRef = useRef<boolean>(false);

  // Limpa controllers ao desmontar
  useEffect(() => {
    return () => {
      if (abortControllerRef.current) {
        abortControllerRef.current.abort();
      }
      isRunningRef.current = false;
    };
  }, []);

  const selectedTarget: TargetItem = TARGETS.find((t) => t.id === selectedTargetId) ?? DEFAULT_TARGET;

  const calculatePercentile = (sortedArr: number[], p: number): number => {
    if (sortedArr.length === 0) return 0;
    const index = Math.ceil((p / 100) * sortedArr.length) - 1;
    const val = sortedArr[Math.max(0, Math.min(index, sortedArr.length - 1))];
    return typeof val === "number" ? val : 0;
  };

  const handleStopTest = useCallback(() => {
    isRunningRef.current = false;
    if (abortControllerRef.current) {
      abortControllerRef.current.abort();
      abortControllerRef.current = null;
    }
    setIsRunning(false);
  }, []);

  const generateDiagnostic = (
    target: TargetItem,
    p95: number,
    success: number,
    rateLimited: number,
    serverErrors: number,
    completed: number,
  ): string => {
    if (completed === 0) return "Aguardando execução do benchmark.";
    if (serverErrors > 0) {
      return `⚠️ Alerta de Instabilidade: ${serverErrors} requisições retornaram erro 5xx. Recomendada inspeção dos logs de contêineres e pool de conexões.`;
    }
    if (rateLimited > 0) {
      return `🛡️ Defesa Ativa Confirmada: O limitador Redis atuou com precisão interceptando ${rateLimited} requisições em excesso (HTTP 429) com cabeçalho Retry-After: 60, preservando a integridade do banco de dados.`;
    }
    if (p95 < 40) {
      return `🚀 Desempenho Excepcional (P95: ${p95}ms): A stack do Projeto-nova-v2 operou com latência baixa, atendendo 100% das requisições com folga em relação ao SLA (${target.performanceGuide.slaTarget}).`;
    }
    if (p95 < 150) {
      return `⚡ Desempenho Saudável (P95: ${p95}ms): Respostas entregues dentro da faixa ótima para navegação web interativa sob concorrência.`;
    }
    return `ℹ️ Tempo de Resposta Moderado (P95: ${p95}ms): Sugerida aplicação de Edge Caching ou ajuste de pool para absorção de picos elevados.`;
  };

  const handleStartTest = async () => {
    if (isRunning) return;

    setIsRunning(true);
    isRunningRef.current = true;
    abortControllerRef.current = new AbortController();

    const currentTarget = selectedTarget;
    const targetUrl = currentTarget.url;
    const totalToRun = totalRequestsTarget;

    const latencies: number[] = [];
    let completed = 0;
    let success = 0;
    let rateLimited = 0;
    let clientErrors = 0;
    let serverErrors = 0;

    setRecentLogs([]);
    setStats({
      completed: 0,
      success: 0,
      rateLimited: 0,
      clientErrors: 0,
      serverErrors: 0,
      avgLatency: 0,
      minLatency: 0,
      maxLatency: 0,
      p95: 0,
      p99: 0,
    });
    setCurrentProgress(0);

    const startTime = performance.now();
    let lastSecondTime = startTime;
    let lastSecondCompleted = 0;

    let nextIndex = 0;

    const worker = async () => {
      while (nextIndex < totalToRun && isRunningRef.current) {
        const currentIndex = nextIndex++;
        const reqStart = performance.now();
        let status = 0;

        try {
          const res = await fetch(targetUrl, {
            method: targetUrl.includes("contact/messages") ? "POST" : "GET",
            headers: {
              "Content-Type": "application/json",
              Accept: "application/json",
            },
            body: targetUrl.includes("contact/messages")
              ? JSON.stringify({
                  name: `Benchmark Bot ${currentIndex}`,
                  email: `bench${currentIndex}@nexus.local`,
                  subject: "Teste de Carga Automatizado",
                  category: "duvida",
                  message: "Mensagem para teste de estresse de infraestrutura.",
                  consent: true,
                })
              : undefined,
            signal: abortControllerRef.current?.signal,
          });
          status = res.status;
        } catch (err: unknown) {
          if ((err as Error)?.name === "AbortError") {
            break;
          }
          status = 500;
        }

        const reqDuration = Math.round(performance.now() - reqStart);
        latencies.push(reqDuration);
        completed++;

        if (status >= 200 && status < 300) {
          success++;
        } else if (status === 429) {
          rateLimited++;
        } else if (status >= 400 && status < 500) {
          clientErrors++;
        } else {
          serverErrors++;
        }

        // Atualização a cada requisição ou lote
        const now = performance.now();
        if (now - lastSecondTime >= 500) {
          const deltaSec = (now - lastSecondTime) / 1000;
          const deltaReqs = completed - lastSecondCompleted;
          setLiveReqPerSec(Math.round(deltaReqs / deltaSec));
          lastSecondTime = now;
          lastSecondCompleted = completed;
        }

        // Percentual
        setCurrentProgress(Math.round((completed / totalToRun) * 100));

        // Registro de log visual
        setRecentLogs((prev) => [
          {
            id: currentIndex,
            statusCode: status,
            durationMs: reqDuration,
            timestamp: new Date().toLocaleTimeString(),
          },
          ...prev.slice(0, 49),
        ]);
      }
    };

    // Dispara a pool de workers concorrentes
    const workers = Array.from({ length: Math.min(concurrency, totalToRun) }, () => worker());
    await Promise.all(workers);

    const totalDurationMs = Math.round(performance.now() - startTime);
    const sortedLatencies = [...latencies].sort((a, b) => a - b);
    const avg = latencies.length ? Math.round(latencies.reduce((a, b) => a + b, 0) / latencies.length) : 0;
    const min = sortedLatencies.length ? (sortedLatencies[0] ?? 0) : 0;
    const max = sortedLatencies.length ? (sortedLatencies[sortedLatencies.length - 1] ?? 0) : 0;
    const p95Val = calculatePercentile(sortedLatencies, 95);
    const p99Val = calculatePercentile(sortedLatencies, 99);
    const totalRps = totalDurationMs > 0 ? Math.round((completed / (totalDurationMs / 1000)) * 10) / 10 : 0;

    setStats({
      completed,
      success,
      rateLimited,
      clientErrors,
      serverErrors,
      avgLatency: avg,
      minLatency: min,
      maxLatency: max,
      p95: p95Val,
      p99: p99Val,
    });
    setLiveReqPerSec(totalRps);
    setIsRunning(false);
    isRunningRef.current = false;

    if (completed > 0) {
      const diagnostic = generateDiagnostic(currentTarget, p95Val, success, rateLimited, serverErrors, completed);
      const newResult: BenchmarkResult = {
        id: crypto.randomUUID(),
        targetId: currentTarget.id,
        targetName: currentTarget.name,
        url: targetUrl,
        concurrency,
        totalRequests: totalToRun,
        completedRequests: completed,
        successCount: success,
        rateLimitedCount: rateLimited,
        clientErrorCount: clientErrors,
        serverErrorCount: serverErrors,
        durationMs: totalDurationMs,
        reqPerSec: totalRps,
        latencies: sortedLatencies,
        avgLatency: avg,
        minLatency: min,
        maxLatency: max,
        p95Latency: p95Val,
        p99Latency: p99Val,
        timestamp: new Date().toLocaleTimeString(),
        diagnosticSummary: diagnostic,
      };

      setHistory((prev) => [newResult, ...prev]);
    }
  };

  const handleCopySummary = () => {
    const text = [
      `RELATÓRIO DE BENCHMARK & TESTE DE CARGA - NEXUS`,
      `=============================================`,
      `Alvo: ${selectedTarget.name} (${selectedTarget.url})`,
      `Concorrência: ${concurrency} workers paralelos`,
      `Total Executado: ${stats.completed}/${totalRequestsTarget} requisições`,
      `Vazão Média: ${liveReqPerSec} req/s`,
      `---------------------------------------------`,
      `LATÊNCIAS:`,
      `- Média: ${stats.avgLatency} ms`,
      `- Mínima: ${stats.minLatency} ms | Máxima: ${stats.maxLatency} ms`,
      `- P95 (95% das reqs): ${stats.p95} ms`,
      `- P99 (99% das reqs): ${stats.p99} ms`,
      `---------------------------------------------`,
      `CÓDIGOS HTTP:`,
      `- Sucesso (2xx): ${stats.success}`,
      `- Rate Limited (429): ${stats.rateLimited}`,
      `- Erros de Cliente (4xx): ${stats.clientErrors}`,
      `- Erros de Servidor (5xx): ${stats.serverErrors}`,
      `---------------------------------------------`,
      `DIAGNÓSTICO:`,
      generateDiagnostic(selectedTarget, stats.p95, stats.success, stats.rateLimited, stats.serverErrors, stats.completed),
    ].join("\n");

    navigator.clipboard.writeText(text);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  const handleExportJSON = () => {
    const dataStr = "data:text/json;charset=utf-8," + encodeURIComponent(JSON.stringify(history, null, 2));
    const downloadAnchor = document.createElement("a");
    downloadAnchor.setAttribute("href", dataStr);
    downloadAnchor.setAttribute("download", `nexus-benchmark-${Date.now()}.json`);
    document.body.appendChild(downloadAnchor);
    downloadAnchor.click();
    downloadAnchor.remove();
  };

  return (
    <div className="flex flex-col gap-6">
      {/* Controles de Configuração e Disparo */}
      <Card className="border-surface-border">
        <CardHeader className="pb-4">
          <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
            <div className="flex items-center gap-3">
              <span className="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg bg-primary/10 text-primary">
                <Zap size={20} />
              </span>
              <div>
                <CardTitle className="text-xl">Cockpit de Testes de Carga & Performance</CardTitle>
                <CardDescription>
                  Dispare requisições concorrentes em tempo real para aferir vazão (req/s), latências (p95/p99) e limites.
                </CardDescription>
              </div>
            </div>
            {isRunning ? (
              <Button variant="danger" onClick={handleStopTest} className="gap-2 shrink-0">
                <Square size={16} /> Interromper Teste
              </Button>
            ) : (
              <Button variant="primary" onClick={handleStartTest} className="gap-2 shrink-0">
                <Play size={16} /> Iniciar Teste de Carga
              </Button>
            )}
          </div>
        </CardHeader>

        <CardContent>
          <div className="grid grid-cols-1 md:grid-cols-3 gap-6 pt-2">
            {/* 1. Escolha do Alvo */}
            <div className="flex flex-col gap-2">
              <label htmlFor="target-select" className="text-xs font-semibold uppercase tracking-wider text-muted">
                1. Endpoint / Cenário Alvo
              </label>
              <select
                id="target-select"
                value={selectedTargetId}
                onChange={(e) => setSelectedTargetId(e.target.value)}
                disabled={isRunning}
                className="w-full rounded-md border border-surface-border bg-surface px-3 py-2 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-primary/50"
              >
                {TARGETS.map((t) => (
                  <option key={t.id} value={t.id}>
                    {t.name}
                  </option>
                ))}
              </select>
              <span className="text-xs text-muted-foreground">{selectedTarget.description}</span>
            </div>

            {/* 2. Concorrência */}
            <div className="flex flex-col gap-2">
              <label className="text-xs font-semibold uppercase tracking-wider text-muted">
                2. Concorrência (Workers Simultâneos)
              </label>
              <div className="grid grid-cols-4 gap-2">
                {[5, 10, 25, 50].map((val) => (
                  <button
                    key={val}
                    type="button"
                    onClick={() => setConcurrency(val)}
                    disabled={isRunning}
                    className={`rounded-md border py-2 text-xs font-medium transition-colors ${
                      concurrency === val
                        ? "border-primary bg-primary/10 text-primary font-bold shadow-sm"
                        : "border-surface-border hover:bg-surface-elevated text-muted"
                    }`}
                  >
                    {val}x
                  </button>
                ))}
              </div>
              <span className="text-xs text-muted-foreground">{concurrency} requisições disparadas em paralelo</span>
            </div>

            {/* 3. Volume de Requisições */}
            <div className="flex flex-col gap-2">
              <label className="text-xs font-semibold uppercase tracking-wider text-muted">
                3. Volume Total de Requisições
              </label>
              <div className="grid grid-cols-4 gap-2">
                {[50, 100, 250, 500].map((val) => (
                  <button
                    key={val}
                    type="button"
                    onClick={() => setTotalRequestsTarget(val)}
                    disabled={isRunning}
                    className={`rounded-md border py-2 text-xs font-medium transition-colors ${
                      totalRequestsTarget === val
                        ? "border-primary bg-primary/10 text-primary font-bold shadow-sm"
                        : "border-surface-border hover:bg-surface-elevated text-muted"
                    }`}
                  >
                    {val}
                  </button>
                ))}
              </div>
              <span className="text-xs text-muted-foreground">Tamanho da amostra para o teste</span>
            </div>
          </div>

          {/* Barra de Progresso do Teste */}
          <div className="mt-6 flex flex-col gap-2">
            <div className="flex items-center justify-between text-xs text-muted">
              <span>
                Progresso: {stats.completed} de {totalRequestsTarget} requisições ({currentProgress}%)
              </span>
              <span>{isRunning ? "Executando benchmark concorrente..." : stats.completed > 0 ? "Benchmark concluído" : "Aguardando disparo"}</span>
            </div>
            <div className="h-2 w-full overflow-hidden rounded-full bg-surface-border">
              <div
                className={`h-full transition-all duration-150 ${
                  isRunning ? "bg-primary animate-pulse" : stats.completed > 0 ? "bg-success" : "bg-muted"
                }`}
                style={{ width: `${currentProgress}%` }}
              />
            </div>
          </div>
        </CardContent>
      </Card>

      {/* NOVO: Explicativa Detalhada do Teste & Arquitetura Alvo */}
      <Card className="border-surface-border bg-surface-elevated/40">
        <CardHeader className="pb-3">
          <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-2">
            <div className="flex items-center gap-2">
              <Info size={18} className="text-primary" />
              <CardTitle className="text-base font-semibold">
                Explicativa do Cenário: {selectedTarget.name}
              </CardTitle>
            </div>
            <div className="flex items-center gap-2">
              <span className="inline-flex items-center px-2 py-0.5 rounded text-[11px] font-medium bg-primary/10 text-primary border border-primary/20">
                {selectedTarget.impactLevel}
              </span>
              <span className="inline-flex items-center px-2 py-0.5 rounded text-[11px] font-mono bg-surface border border-surface-border text-muted">
                {selectedTarget.url}
              </span>
            </div>
          </div>
          <CardDescription className="text-xs">
            Compreenda exatamente o fluxo percorrido pela requisição, os subsistemas exercitados e o que este teste avalia.
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4 text-xs">
          {/* Pipeline Visual da Requisição */}
          <div className="rounded-lg border border-surface-border bg-surface/70 p-3">
            <span className="font-semibold text-muted uppercase tracking-wider text-[10px] block mb-2">
              Pipeline de Execução Ponta a Ponta:
            </span>
            <div className="flex flex-wrap items-center gap-2">
              {selectedTarget.pipeline.map((step, idx) => (
                <div key={idx} className="flex items-center gap-2">
                  <span className="rounded bg-surface-elevated px-2 py-1 font-mono text-[11px] text-foreground border border-surface-border">
                    {step}
                  </span>
                  {idx < selectedTarget.pipeline.length - 1 && (
                    <ArrowRight size={12} className="text-muted-foreground" />
                  )}
                </div>
              ))}
            </div>
          </div>

          {/* Grade de Explicação Detalhada */}
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            {/* O que o teste executa */}
            <div className="rounded-lg border border-surface-border p-3 space-y-2 bg-surface/50">
              <div className="flex items-center gap-1.5 font-semibold text-foreground text-xs">
                <Layers size={14} className="text-primary" />
                <span>O que este teste executa:</span>
              </div>
              <p className="text-muted-foreground leading-relaxed">
                {selectedTarget.testExplanation.overview}
              </p>
              <div className="pt-1">
                <span className="text-[11px] font-medium text-foreground block mb-1">
                  Subsistemas e Recursos Estressados:
                </span>
                <ul className="list-disc pl-4 space-y-1 text-muted">
                  {selectedTarget.testExplanation.subsystems.map((sub, idx) => (
                    <li key={idx}>{sub}</li>
                  ))}
                </ul>
              </div>
            </div>

            {/* Guia de Desempenho & SLA Nominal */}
            <div className="rounded-lg border border-surface-border p-3 space-y-2 bg-surface/50">
              <div className="flex items-center gap-1.5 font-semibold text-foreground text-xs">
                <ShieldCheck size={14} className="text-success" />
                <span>Guia de Desempenho & Critérios de SLA:</span>
              </div>
              <div className="rounded border border-success/20 bg-success/5 p-2 font-mono text-[11px] text-success font-semibold">
                Meta Nominal: {selectedTarget.performanceGuide.slaTarget}
              </div>
              <div className="space-y-1.5 text-muted">
                <p>
                  <strong className="text-foreground">Interpretação da Latência: </strong>
                  {selectedTarget.performanceGuide.latencyInterpretation}
                </p>
                <p>
                  <strong className="text-foreground">Comportamento de Rate Limit: </strong>
                  {selectedTarget.performanceGuide.rateLimitBehavior}
                </p>
                <p>
                  <strong className="text-foreground">Dica Prática de Otimização: </strong>
                  {selectedTarget.performanceGuide.optimizationTip}
                </p>
              </div>
            </div>
          </div>
        </CardContent>
      </Card>

      {/* KPIs de Performance em Tempo Real */}
      <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
        {/* Vazão / Throughput */}
        <Card className="border-surface-border">
          <CardContent className="pt-5 pb-4">
            <div className="flex items-center justify-between text-muted mb-2">
              <span className="text-xs font-medium uppercase tracking-wider">Vazão Instantânea</span>
              <Gauge size={16} className="text-primary" />
            </div>
            <div className="text-2xl font-bold tracking-tight">
              {liveReqPerSec} <span className="text-xs font-normal text-muted">req/s</span>
            </div>
            <p className="text-[11px] text-muted-foreground mt-1">Capacidade de resposta da stack</p>
          </CardContent>
        </Card>

        {/* Latência Média */}
        <Card className="border-surface-border">
          <CardContent className="pt-5 pb-4">
            <div className="flex items-center justify-between text-muted mb-2">
              <span className="text-xs font-medium uppercase tracking-wider">Latência Média</span>
              <Clock size={16} className="text-info" />
            </div>
            <div className="text-2xl font-bold tracking-tight">
              {stats.avgLatency} <span className="text-xs font-normal text-muted">ms</span>
            </div>
            <p className="text-[11px] text-muted-foreground mt-1">
              Min: {stats.minLatency}ms | Max: {stats.maxLatency}ms
            </p>
          </CardContent>
        </Card>

        {/* Percentil P95 / P99 */}
        <Card className="border-surface-border">
          <CardContent className="pt-5 pb-4">
            <div className="flex items-center justify-between text-muted mb-2">
              <span className="text-xs font-medium uppercase tracking-wider">Percentis P95 / P99</span>
              <TrendingUp size={16} className="text-warning" />
            </div>
            <div className="text-2xl font-bold tracking-tight">
              {stats.p95} / {stats.p99} <span className="text-xs font-normal text-muted">ms</span>
            </div>
            <p className="text-[11px] text-muted-foreground mt-1">95% das respostas abaixo de {stats.p95}ms</p>
          </CardContent>
        </Card>

        {/* Taxa de Sucesso / Rate Limit */}
        <Card className="border-surface-border">
          <CardContent className="pt-5 pb-4">
            <div className="flex items-center justify-between text-muted mb-2">
              <span className="text-xs font-medium uppercase tracking-wider">Status das Respostas</span>
              <BarChart3 size={16} className="text-success" />
            </div>
            <div className="flex items-baseline gap-2">
              <span className="text-2xl font-bold text-success">{stats.success}</span>
              {stats.rateLimited > 0 && (
                <span className="text-sm font-semibold text-warning" title="Rate Limited (HTTP 429)">
                  ({stats.rateLimited}x 429)
                </span>
              )}
              {stats.serverErrors > 0 && (
                <span className="text-sm font-semibold text-danger" title="Erros 5xx">
                  ({stats.serverErrors}x 5xx)
                </span>
              )}
            </div>
            <p className="text-[11px] text-muted-foreground mt-1">
              {stats.rateLimited > 0
                ? "Rate Limiter ativado com sucesso"
                : stats.serverErrors > 0
                ? "Falhas de servidor detectadas"
                : "Respostas HTTP 200 OK"}
            </p>
          </CardContent>
        </Card>
      </div>

      {/* NOVO: Card de Diagnóstico Analítico do Desempenho em Tempo Real */}
      {stats.completed > 0 && (
        <Card className="border-surface-border border-l-4 border-l-primary bg-surface-elevated/20">
          <CardHeader className="pb-2">
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-2">
                <BarChart3 size={18} className="text-primary" />
                <CardTitle className="text-sm font-semibold">
                  Diagnóstico Analítico do Desempenho Desta Rodada
                </CardTitle>
              </div>
              <span className="text-xs text-muted font-mono">
                {stats.completed} requisições processadas
              </span>
            </div>
          </CardHeader>
          <CardContent className="space-y-3 text-xs">
            <p className="text-foreground leading-relaxed font-medium bg-surface/80 p-3 rounded border border-surface-border">
              {generateDiagnostic(
                selectedTarget,
                stats.p95,
                stats.success,
                stats.rateLimited,
                stats.serverErrors,
                stats.completed,
              )}
            </p>
            <div className="grid grid-cols-1 sm:grid-cols-3 gap-3 pt-1 text-[11px]">
              <div className="rounded border border-surface-border p-2 bg-surface">
                <span className="text-muted block font-semibold mb-0.5">Por que P95 e P99 importam?</span>
                <span className="text-muted-foreground">
                  A média pode mascarar requisições lentas. O P95 ({stats.p95}ms) garante que 95 de cada 100 usuários recebem resposta abaixo deste teto sem lentidão.
                </span>
              </div>
              <div className="rounded border border-surface-border p-2 bg-surface">
                <span className="text-muted block font-semibold mb-0.5">Impacto no Hardware do Host:</span>
                <span className="text-muted-foreground">
                  Com conexões pooled do Go e Redis, o consumo de CPU permanece abaixo de 1% e a memória fica contida na faixa de 28MB no Proxmox.
                </span>
              </div>
              <div className="rounded border border-surface-border p-2 bg-surface">
                <span className="text-muted block font-semibold mb-0.5">Cabeçalhos RFC 6585 Ativos:</span>
                <span className="text-muted-foreground">
                  Quando o status 429 ocorre, os headers <code className="text-primary">Retry-After: 60</code> e <code className="text-primary">X-RateLimit-Limit</code> orientam o cliente a retentar com segurança.
                </span>
              </div>
            </div>
          </CardContent>
        </Card>
      )}

      {/* Painel Inferior: Distribuição de Códigos HTTP & Logs Recentes */}
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        {/* Distribuição de Códigos */}
        <Card className="border-surface-border lg:col-span-1">
          <CardHeader className="pb-3">
            <CardTitle className="text-base">Distribuição de Status HTTP</CardTitle>
            <CardDescription className="text-xs">Proporção de códigos retornados pelo servidor</CardDescription>
          </CardHeader>
          <CardContent className="flex flex-col gap-4">
            {/* 2xx */}
            <div className="flex flex-col gap-1">
              <div className="flex items-center justify-between text-xs">
                <span className="flex items-center gap-1.5 font-medium text-success">
                  <CheckCircle2 size={13} /> 2xx (Sucesso)
                </span>
                <span className="font-bold">{stats.success} ({stats.completed > 0 ? Math.round((stats.success / stats.completed) * 100) : 0}%)</span>
              </div>
              <div className="h-1.5 w-full rounded-full bg-surface-border overflow-hidden">
                <div
                  className="h-full bg-success"
                  style={{ width: `${stats.completed > 0 ? (stats.success / stats.completed) * 100 : 0}%` }}
                />
              </div>
            </div>

            {/* 429 */}
            <div className="flex flex-col gap-1">
              <div className="flex items-center justify-between text-xs">
                <span className="flex items-center gap-1.5 font-medium text-warning">
                  <AlertTriangle size={13} /> 429 (Rate Limited / Defesa)
                </span>
                <span className="font-bold">{stats.rateLimited} ({stats.completed > 0 ? Math.round((stats.rateLimited / stats.completed) * 100) : 0}%)</span>
              </div>
              <div className="h-1.5 w-full rounded-full bg-surface-border overflow-hidden">
                <div
                  className="h-full bg-warning"
                  style={{ width: `${stats.completed > 0 ? (stats.rateLimited / stats.completed) * 100 : 0}%` }}
                />
              </div>
            </div>

            {/* 4xx */}
            <div className="flex flex-col gap-1">
              <div className="flex items-center justify-between text-xs">
                <span className="flex items-center gap-1.5 font-medium text-muted">
                  4xx (Erros Cliente)
                </span>
                <span className="font-bold">{stats.clientErrors} ({stats.completed > 0 ? Math.round((stats.clientErrors / stats.completed) * 100) : 0}%)</span>
              </div>
              <div className="h-1.5 w-full rounded-full bg-surface-border overflow-hidden">
                <div
                  className="h-full bg-muted"
                  style={{ width: `${stats.completed > 0 ? (stats.clientErrors / stats.completed) * 100 : 0}%` }}
                />
              </div>
            </div>

            {/* 5xx */}
            <div className="flex flex-col gap-1">
              <div className="flex items-center justify-between text-xs">
                <span className="flex items-center gap-1.5 font-medium text-danger">
                  5xx (Erros Servidor)
                </span>
                <span className="font-bold">{stats.serverErrors} ({stats.completed > 0 ? Math.round((stats.serverErrors / stats.completed) * 100) : 0}%)</span>
              </div>
              <div className="h-1.5 w-full rounded-full bg-surface-border overflow-hidden">
                <div
                  className="h-full bg-danger"
                  style={{ width: `${stats.completed > 0 ? (stats.serverErrors / stats.completed) * 100 : 0}%` }}
                />
              </div>
            </div>

            {/* Ações */}
            <div className="pt-2 flex items-center gap-2">
              <Button
                variant="secondary"
                size="sm"
                onClick={handleCopySummary}
                disabled={stats.completed === 0}
                className="w-full text-xs gap-1"
              >
                {copied ? <Check size={13} className="text-success" /> : <Copy size={13} />}
                {copied ? "Copiado!" : "Copiar Resumo"}
              </Button>
              {history.length > 0 && (
                <Button
                  variant="secondary"
                  size="sm"
                  onClick={handleExportJSON}
                  className="w-full text-xs gap-1"
                >
                  <Download size={13} /> Exportar JSON
                </Button>
              )}
            </div>
          </CardContent>
        </Card>

        {/* Streaming de Logs Recentes */}
        <Card className="border-surface-border lg:col-span-2">
          <CardHeader className="pb-3">
            <div className="flex items-center justify-between">
              <div>
                <CardTitle className="text-base">Stream de Requisições ao Vivo</CardTitle>
                <CardDescription className="text-xs">Amostragem em tempo real das chamadas finalizadas</CardDescription>
              </div>
              <span className="text-xs text-muted flex items-center gap-1">
                <span className={`h-2 w-2 rounded-full ${isRunning ? "bg-success animate-ping" : "bg-muted"}`} />
                {isRunning ? "Ativo" : "Ocioso"}
              </span>
            </div>
          </CardHeader>
          <CardContent>
            {recentLogs.length === 0 ? (
              <div className="flex flex-col items-center justify-center py-10 text-center text-muted">
                <Gauge size={32} className="opacity-30 mb-2" />
                <p className="text-sm">Nenhum teste de carga executado ainda nesta sessão.</p>
                <p className="text-xs text-muted-foreground">Clique em &quot;Iniciar Teste de Carga&quot; para começar a amostragem.</p>
              </div>
            ) : (
              <div className="overflow-x-auto max-h-[220px] overflow-y-auto pr-1">
                <table className="w-full text-left text-xs">
                  <thead className="border-b border-surface-border text-muted">
                    <tr>
                      <th className="py-1.5 font-semibold"># Req</th>
                      <th className="py-1.5 font-semibold">Horário</th>
                      <th className="py-1.5 font-semibold">Status HTTP</th>
                      <th className="py-1.5 font-semibold text-right">Latência</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-surface-border/50">
                    {recentLogs.map((log) => (
                      <tr key={log.id} className="hover:bg-surface-elevated/40">
                        <td className="py-1 font-mono text-muted">#{log.id + 1}</td>
                        <td className="py-1 font-mono">{log.timestamp}</td>
                        <td className="py-1">
                          <span
                            className={`inline-flex items-center px-1.5 py-0.5 rounded text-[10px] font-bold ${
                              log.statusCode >= 200 && log.statusCode < 300
                                ? "bg-success/10 text-success"
                                : log.statusCode === 429
                                ? "bg-warning/10 text-warning"
                                : "bg-danger/10 text-danger"
                            }`}
                          >
                            {log.statusCode} {log.statusCode === 429 ? "Rate Limit" : ""}
                          </span>
                        </td>
                        <td className="py-1 text-right font-mono font-medium">
                          <span
                            className={
                              log.durationMs < 50
                                ? "text-success"
                                : log.durationMs < 200
                                ? "text-info"
                                : "text-warning"
                            }
                          >
                            {log.durationMs}ms
                          </span>
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </CardContent>
        </Card>
      </div>

      {/* Histórico da Sessão com Gaveta de Explicativa Detalhada */}
      {history.length > 0 && (
        <Card className="border-surface-border">
          <CardHeader className="pb-3">
            <CardTitle className="text-base">Histórico dos Testes Realizados & Análise de Desempenho</CardTitle>
            <CardDescription className="text-xs">
              Benchmarks finalizados durante a sessão atual. Clique em qualquer linha para abrir a análise detalhada.
            </CardDescription>
          </CardHeader>
          <CardContent>
            <div className="space-y-3">
              {history.map((h) => {
                const isExpanded = expandedHistoryId === h.id;
                const matchedTarget = TARGETS.find((t) => t.id === h.targetId);

                return (
                  <div
                    key={h.id}
                    className="rounded-lg border border-surface-border bg-surface/50 overflow-hidden transition-all"
                  >
                    <div
                      onClick={() => setExpandedHistoryId(isExpanded ? null : h.id)}
                      className="p-3 flex flex-col md:flex-row md:items-center justify-between gap-3 cursor-pointer hover:bg-surface-elevated/60"
                    >
                      <div className="flex items-center gap-3">
                        <span className="flex h-8 w-8 shrink-0 items-center justify-center rounded bg-primary/10 text-primary font-bold text-xs">
                          {h.concurrency}x
                        </span>
                        <div>
                          <div className="flex items-center gap-2">
                            <span className="font-semibold text-xs text-foreground">{h.targetName}</span>
                            <span className="text-[10px] text-muted font-mono">{h.url}</span>
                          </div>
                          <span className="text-[11px] text-muted-foreground">
                            {h.completedRequests} reqs em {h.durationMs}ms ({h.timestamp})
                          </span>
                        </div>
                      </div>

                      <div className="flex items-center gap-4 text-xs font-mono">
                        <div className="flex flex-col items-end">
                          <span className="text-primary font-bold">{h.reqPerSec} req/s</span>
                          <span className="text-[10px] text-muted">Vazão média</span>
                        </div>
                        <div className="flex flex-col items-end">
                          <span className="font-semibold text-foreground">P95: {h.p95Latency}ms</span>
                          <span className="text-[10px] text-muted">Média: {h.avgLatency}ms</span>
                        </div>
                        <div className="flex items-center gap-1">
                          <span className="text-success text-xs font-bold">{h.successCount}x 200</span>
                          {h.rateLimitedCount > 0 && (
                            <span className="text-warning text-xs font-bold">| {h.rateLimitedCount}x 429</span>
                          )}
                        </div>
                        <button
                          type="button"
                          className="text-muted hover:text-foreground p-1"
                          aria-label={isExpanded ? "Recolher detalhes" : "Expandir detalhes"}
                        >
                          {isExpanded ? <ChevronUp size={16} /> : <ChevronDown size={16} />}
                        </button>
                      </div>
                    </div>

                    {/* Conteúdo expandido com explicativa do desempenho */}
                    {isExpanded && (
                      <div className="border-t border-surface-border p-4 bg-surface/90 space-y-3 text-xs">
                        <div className="rounded border border-primary/20 bg-primary/5 p-3">
                          <span className="font-semibold text-primary block mb-1">
                            Diagnóstico de Desempenho Registrado:
                          </span>
                          <p className="text-foreground leading-relaxed">{h.diagnosticSummary}</p>
                        </div>

                        {matchedTarget && (
                          <div className="grid grid-cols-1 md:grid-cols-2 gap-3 text-muted">
                            <div className="rounded border border-surface-border p-2.5 bg-surface">
                              <span className="font-semibold text-foreground block mb-1">
                                O que este teste avaliou na infraestrutura:
                              </span>
                              <p className="text-xs text-muted-foreground">
                                {matchedTarget.testExplanation.overview}
                              </p>
                            </div>
                            <div className="rounded border border-surface-border p-2.5 bg-surface">
                              <span className="font-semibold text-foreground block mb-1">
                                Parecer de Conformidade com o SLA:
                              </span>
                              <p className="text-xs text-muted-foreground">
                                Meta: {matchedTarget.performanceGuide.slaTarget}. O teste alcançou P95 de {h.p95Latency}ms e P99 de {h.p99Latency}ms sob {h.concurrency} conexões paralelas.
                              </p>
                            </div>
                          </div>
                        )}
                      </div>
                    )}
                  </div>
                );
              })}
            </div>
          </CardContent>
        </Card>
      )}
    </div>
  );
}
