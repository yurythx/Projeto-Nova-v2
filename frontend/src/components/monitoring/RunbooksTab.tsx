"use client";

import { useState } from "react";
import {
  ShieldAlert,
  FileCode2,
  Users,
  Copy,
  Check,
  ExternalLink,
  Layers,
  Archive,
  FolderLock,
  Radio,
  FileText,
} from "lucide-react";
import { Button } from "@/components/ui/Button";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/Card";

export function RunbooksTab() {
  const [copiedCmd, setCopiedCmd] = useState<string | null>(null);

  const copyToClipboard = (text: string, id: string) => {
    navigator.clipboard.writeText(text);
    setCopiedCmd(id);
    setTimeout(() => setCopiedCmd(null), 2000);
  };

  const personas = [
    {
      username: "iam.admin",
      name: "Administração de identidade",
      role: "Núcleo IAM",
      unit: "Plataforma",
      profiles: "Administrador",
      permissions: "iam:manage (usuários, perfis e escopo)",
    },
    {
      username: "auditoria.leitor",
      name: "Leitura da trilha",
      role: "Núcleo Auditoria",
      unit: "Plataforma",
      profiles: "Auditor",
      permissions: "auditoria:read",
    },
    {
      username: "plugin.operador",
      name: "Operação de um plug-in",
      role: "Módulo desligado por padrão",
      unit: "Órgão no escopo",
      profiles: "Operador",
      permissions: "as do plug-in, só depois de ligado",
    },
  ];

  const topologyServices = [
    { name: "Caddy Reverse Proxy", ports: "80, 443 (HTTP/2 & HTTP/3)", desc: "TLS automático, roteamento de SNI e terminação HTTPS com CA corporativa." },
    { name: "Next.js 16 BFF & Frontend", ports: "3000 (interno) / 127.0.0.1:3010", desc: "SSR, React Compiler, PWA, Service Worker e Middleware de segurança." },
    { name: "Go 1.26 Backend REST Core", ports: "8000 (interno) / 8443 (HTTPS)", desc: "Clean Architecture, Chi router, JWT local, autenticação e WebSocket /ws." },
    { name: "PostgreSQL 16 Engine", ports: "5432 (interno)", desc: "Banco relacional ACID, Goose migrations e fila transacional de Outbox." },
    { name: "MinIO S3 Storage", ports: "9000 (interno) / 9443 (HTTPS)", desc: "Bucket WORM de auditoria e repositório de documentos/anexos." },
    { name: "Redis 7.4 Alpine", ports: "6379 (interno)", desc: "Cache IAM, lockout progressivo de login e rate limiting de endpoints." },
    { name: "RabbitMQ 3.13 AMQP", ports: "5672 (AMQP) / 15672 (Admin)", desc: "Message broker orientado a eventos para workers assíncronos." },
    { name: "Keycloak 26 OIDC", ports: "8080 (interno) / 8543 (HTTPS)", desc: "Single Sign-On (SSO) corporativo e emissor de tokens federados." },
  ];

  return (
    <div className="flex flex-col gap-8">
      {/* 1. DISASTER RECOVERY & BACKUP */}
      <Card className="border-surface-border">
        <CardHeader className="pb-4">
          <div className="flex items-center gap-3">
            <span className="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg bg-danger/10 text-danger">
              <ShieldAlert size={20} />
            </span>
            <div>
              <CardTitle className="text-xl">Disaster Recovery (DR) & Procedimentos de Restauração</CardTitle>
              <CardDescription>
                Diretrizes de recuperação a frio (Cold Standby) e continuidade do Projeto-nova-v2.
              </CardDescription>
            </div>
          </div>
        </CardHeader>

        <CardContent className="flex flex-col gap-6">
          {/* Status do DR */}
          <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
            <div className="p-3.5 rounded-lg border border-surface-border bg-surface-elevated/40">
              <span className="text-[11px] font-semibold uppercase tracking-wider text-muted">Diretório de Snapshots</span>
              <p className="font-mono text-sm font-bold text-foreground mt-1">/root/backups/nova</p>
              <span className="text-xs text-muted-foreground">Isolado com umask 077</span>
            </div>
            <div className="p-3.5 rounded-lg border border-surface-border bg-surface-elevated/40">
              <span className="text-[11px] font-semibold uppercase tracking-wider text-muted">Agendamento Diário</span>
              <p className="text-sm font-bold text-foreground mt-1">02:30 AM (Madrugada)</p>
              <span className="text-xs text-muted-foreground">Configurado em /etc/cron.d/nova-backup</span>
            </div>
            <div className="p-3.5 rounded-lg border border-surface-border bg-surface-elevated/40">
              <span className="text-[11px] font-semibold uppercase tracking-wider text-muted">Política de Retenção</span>
              <p className="text-sm font-bold text-foreground mt-1">7 Dias Locais</p>
              <span className="text-xs text-muted-foreground">Limpeza automática de snapshots antigos</span>
            </div>
            <div className="p-3.5 rounded-lg border border-surface-border bg-surface-elevated/40">
              <span className="text-[11px] font-semibold uppercase tracking-wider text-muted">Integridade Criptográfica</span>
              <p className="text-sm font-bold text-success mt-1">SHA-256 Verificado</p>
              <span className="text-xs text-muted-foreground">Checksum automático de todos os arquivos</span>
            </div>
          </div>

          {/* Comandos Rápidos */}
          <div className="flex flex-col gap-3">
            <h3 className="text-sm font-bold uppercase tracking-wider text-foreground">Comandos Operacionais de Produção</h3>

            {/* Comando 1: Verificação sem alterar banco */}
            <div className="flex flex-col gap-1.5 p-3 rounded-lg border border-surface-border bg-surface">
              <div className="flex items-center justify-between">
                <span className="text-xs font-semibold text-foreground flex items-center gap-1.5">
                  <Archive size={14} className="text-primary" /> 1. Testar Restauração sem Alterar Produção (--verify)
                </span>
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={() =>
                    copyToClipboard(
                      "./scripts/restore.sh --verify $(ls -td /root/backups/nova/* | head -1)",
                      "cmd-verify"
                    )
                  }
                  className="h-7 px-2 text-xs gap-1"
                >
                  {copiedCmd === "cmd-verify" ? <Check size={12} className="text-success" /> : <Copy size={12} />}
                  {copiedCmd === "cmd-verify" ? "Copiado!" : "Copiar"}
                </Button>
              </div>
              <p className="text-xs text-muted">
                Restaura o último dump num banco temporário (<code>nova_restore_check</code>), confere contagens de usuários, unidades e auditoria e descarta o temporário.
              </p>
              <pre className="p-2 rounded bg-surface-elevated font-mono text-xs text-foreground overflow-x-auto">
                ./scripts/restore.sh --verify $(ls -td /root/backups/nova/* | head -1)
              </pre>
            </div>

            {/* Comando 2: Restauração Real de Desastre */}
            <div className="flex flex-col gap-1.5 p-3 rounded-lg border border-surface-border bg-surface">
              <div className="flex items-center justify-between">
                <span className="text-xs font-semibold text-danger flex items-center gap-1.5">
                  <FolderLock size={14} /> 2. Restauração Completa de Desastre (--yes com MinIO)
                </span>
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={() =>
                    copyToClipboard(
                      "./scripts/restore.sh --yes /root/backups/nova/<DATA-HORA> --minio",
                      "cmd-restore"
                    )
                  }
                  className="h-7 px-2 text-xs gap-1"
                >
                  {copiedCmd === "cmd-restore" ? <Check size={12} className="text-success" /> : <Copy size={12} />}
                  {copiedCmd === "cmd-restore" ? "Copiado!" : "Copiar"}
                </Button>
              </div>
              <p className="text-xs text-muted">
                Para a API e workers, substitui o banco pelo backup e sobrepõe os objetos de armazenamento S3.
              </p>
              <pre className="p-2 rounded bg-surface-elevated font-mono text-xs text-danger overflow-x-auto">
                ./scripts/restore.sh --yes /root/backups/nova/&lt;DATA-HORA&gt; --minio
              </pre>
            </div>
          </div>
        </CardContent>
      </Card>

      {/* 2. MATRIZ DE IDENTIDADE & PERFIS (IAM) */}
      <Card className="border-surface-border">
        <CardHeader className="pb-4">
          <div className="flex items-center gap-3">
            <span className="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg bg-primary/10 text-primary">
              <Users size={20} />
            </span>
            <div>
              <CardTitle className="text-xl">Matriz de Identidade & Acessos Homologados (IAM)</CardTitle>
              <CardDescription>
                Perfis do núcleo (IAM e Auditoria), com escopo por órgão (ADR 013).
              </CardDescription>
            </div>
          </div>
        </CardHeader>
        <CardContent>
          <div className="overflow-x-auto">
            <table className="w-full text-left text-xs">
              <thead className="border-b border-surface-border text-muted font-semibold uppercase tracking-wider">
                <tr>
                  <th className="py-2.5">Persona / Usuário</th>
                  <th className="py-2.5">Cargo Institucional</th>
                  <th className="py-2.5">Lotação (Unidade)</th>
                  <th className="py-2.5">Perfis de Acesso</th>
                  <th className="py-2.5">Permissões Efetivas</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-surface-border/50">
                {personas.map((p) => (
                  <tr key={p.username} className="hover:bg-surface-elevated/40">
                    <td className="py-2.5 font-medium">
                      <div className="flex flex-col">
                        <span className="font-bold text-foreground">{p.name}</span>
                        <span className="font-mono text-[11px] text-muted">{p.username}</span>
                      </div>
                    </td>
                    <td className="py-2.5 text-muted">{p.role}</td>
                    <td className="py-2.5 font-medium text-foreground">{p.unit}</td>
                    <td className="py-2.5">
                      <span className="inline-flex items-center px-2 py-0.5 rounded text-[11px] font-semibold bg-primary/10 text-primary">
                        {p.profiles}
                      </span>
                    </td>
                    <td className="py-2.5 font-mono text-[11px] text-muted-foreground">{p.permissions}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </CardContent>
      </Card>

      {/* 3. TOPOLOGIA DE SERVIÇOS & DOCUMENTAÇÃO TÉCNICA */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* Topologia dos Microsserviços */}
        <Card className="border-surface-border">
          <CardHeader className="pb-3">
            <div className="flex items-center gap-2">
              <Layers size={18} className="text-primary" />
              <CardTitle className="text-base">Topologia da Infraestrutura Docker</CardTitle>
            </div>
            <CardDescription className="text-xs">Portas, redes internas e finalidade de cada serviço da stack</CardDescription>
          </CardHeader>
          <CardContent className="flex flex-col gap-3">
            <div className="overflow-x-auto">
              <table className="w-full text-left text-xs">
                <thead className="border-b border-surface-border text-muted font-semibold">
                  <tr>
                    <th className="py-2">Serviço</th>
                    <th className="py-2">Portas</th>
                    <th className="py-2">Função Arquitetural</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-surface-border/50">
                  {topologyServices.map((s) => (
                    <tr key={s.name} className="hover:bg-surface-elevated/40">
                      <td className="py-2 font-bold text-foreground">{s.name}</td>
                      <td className="py-2 font-mono text-[11px] text-muted">{s.ports}</td>
                      <td className="py-2 text-[11px] text-muted-foreground">{s.desc}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </CardContent>
        </Card>

        {/* Links Rápidos de Handover & OpenAPI */}
        <Card className="border-surface-border">
          <CardHeader className="pb-3">
            <div className="flex items-center gap-2">
              <FileCode2 size={18} className="text-primary" />
              <CardTitle className="text-base">Catálogo de APIs & Documentação Interativa</CardTitle>
            </div>
            <CardDescription className="text-xs">Acesso rápido aos contratos e especificações técnicas de integração</CardDescription>
          </CardHeader>
          <CardContent className="flex flex-col gap-4">
            <a
              href="https://192.168.1.42:8443/docs"
              target="_blank"
              rel="noreferrer"
              className="flex items-center justify-between p-3 rounded-lg border border-surface-border hover:border-primary/50 hover:bg-surface-elevated transition-colors"
            >
              <div className="flex items-center gap-3">
                <div className="flex h-9 w-9 items-center justify-center rounded bg-primary/10 text-primary">
                  <FileText size={18} />
                </div>
                <div>
                  <h4 className="text-sm font-bold text-foreground">Swagger UI Interativo (/docs)</h4>
                  <p className="text-xs text-muted">Navegue e teste endpoints REST protegidos e públicos com Bearer Auth</p>
                </div>
              </div>
              <ExternalLink size={14} className="text-muted" />
            </a>

            <a
              href="https://192.168.1.42:8443/openapi.json"
              target="_blank"
              rel="noreferrer"
              className="flex items-center justify-between p-3 rounded-lg border border-surface-border hover:border-primary/50 hover:bg-surface-elevated transition-colors"
            >
              <div className="flex items-center gap-3">
                <div className="flex h-9 w-9 items-center justify-center rounded bg-primary/10 text-primary">
                  <FileCode2 size={18} />
                </div>
                <div>
                  <h4 className="text-sm font-bold text-foreground">Especificação OpenAPI 3.0 (openapi.json)</h4>
                  <p className="text-xs text-muted">Contrato de esquemas, validações JSON Schema e DTOs da plataforma</p>
                </div>
              </div>
              <ExternalLink size={14} className="text-muted" />
            </a>

            <a
              href="https://192.168.1.42:8443/metrics"
              target="_blank"
              rel="noreferrer"
              className="flex items-center justify-between p-3 rounded-lg border border-surface-border hover:border-primary/50 hover:bg-surface-elevated transition-colors"
            >
              <div className="flex items-center gap-3">
                <div className="flex h-9 w-9 items-center justify-center rounded bg-primary/10 text-primary">
                  <Radio size={18} />
                </div>
                <div>
                  <h4 className="text-sm font-bold text-foreground">Endpoint Prometheus (/metrics)</h4>
                  <p className="text-xs text-muted">Telemetria de contadores HTTP, latência e estatísticas do runtime Go</p>
                </div>
              </div>
              <ExternalLink size={14} className="text-muted" />
            </a>

            <div className="p-3 rounded-lg bg-surface-elevated/40 border border-surface-border text-xs text-muted">
              <span className="font-semibold text-foreground">Nota de Governança:</span> A autenticação nas chamadas Swagger requer o token JWT emitido pelo Keycloak ou login local (`/api/v1/auth/login`).
            </div>
          </CardContent>
        </Card>
      </div>
    </div>
  );
}
