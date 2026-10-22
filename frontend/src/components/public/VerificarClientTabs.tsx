"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { KeyRound, QrCode, FileCheck, Shield, BookOpen, ExternalLink } from "lucide-react";
import { QRScannerTab } from "./QRScannerTab";
import { FileHash } from "@/components/signum/FileHash";

interface VerificarClientTabsProps {
  children: React.ReactNode;
}

export function VerificarClientTabs({ children }: VerificarClientTabsProps) {
  const router = useRouter();
  const [activeTab, setActiveTab] = useState<"manual" | "qrcode" | "file">("manual");
  const [fileHash, setFileHash] = useState<string | null>(null);

  return (
    <div className="flex flex-col gap-6">
      {/* Seletor de Métodos */}
      <div className="flex flex-wrap gap-2 rounded-xl border border-surface-border bg-surface-subtle p-1.5" role="tablist" aria-label="Métodos de Verificação">
        <button
          type="button"
          role="tab"
          aria-selected={activeTab === "manual"}
          onClick={() => setActiveTab("manual")}
          className={`flex items-center gap-2 rounded-lg px-4 py-2 text-xs font-medium transition-all ${
            activeTab === "manual"
              ? "bg-surface shadow-xs text-foreground"
              : "text-muted hover:text-foreground"
          }`}
        >
          <KeyRound size={15} aria-hidden="true" />
          <span>Digitar Código (UUID)</span>
        </button>

        <button
          type="button"
          role="tab"
          aria-selected={activeTab === "qrcode"}
          onClick={() => setActiveTab("qrcode")}
          className={`flex items-center gap-2 rounded-lg px-4 py-2 text-xs font-medium transition-all ${
            activeTab === "qrcode"
              ? "bg-surface shadow-xs text-foreground"
              : "text-muted hover:text-foreground"
          }`}
        >
          <QrCode size={15} aria-hidden="true" />
          <span>Escanear QR Code</span>
        </button>

        <button
          type="button"
          role="tab"
          aria-selected={activeTab === "file"}
          onClick={() => setActiveTab("file")}
          className={`flex items-center gap-2 rounded-lg px-4 py-2 text-xs font-medium transition-all ${
            activeTab === "file"
              ? "bg-surface shadow-xs text-foreground"
              : "text-muted hover:text-foreground"
          }`}
        >
          <FileCheck size={15} aria-hidden="true" />
          <span>Verificar Integridade por Arquivo</span>
        </button>
      </div>

      {/* Conteúdo das Abas */}
      {activeTab === "manual" && (
        <div className="flex flex-col gap-4 animate-fadeIn">
          {children}
        </div>
      )}

      {activeTab === "qrcode" && (
        <div className="flex flex-col gap-4 animate-fadeIn">
          <QRScannerTab
            onCodeDetected={(code) => {
              router.push(`/verificar/${code}`);
            }}
          />
        </div>
      )}

      {activeTab === "file" && (
        <div className="flex flex-col gap-4 rounded-xl border border-surface-border bg-surface-raised p-5 text-sm animate-fadeIn">
          <div className="flex flex-col gap-1">
            <h3 className="font-semibold text-foreground">Conferência Criptográfica de Arquivo</h3>
            <p className="text-xs text-muted">
              Selecione o arquivo em PDF ou documento para calcular instantaneamente sua impressão digital SHA-256 no navegador:
            </p>
          </div>
          <FileHash
            id="verificar-arquivo-local"
            label="Escolher documento para calcular hash"
            onHash={(hash) => {
              setFileHash(hash);
            }}
          />
          {fileHash && (
            <div className="flex flex-col gap-2 rounded-lg border border-primary/20 bg-primary/5 p-3 text-xs">
              <span className="font-medium text-primary">Impressão Digital Calculada com Sucesso:</span>
              <code className="break-all font-mono text-[11px] select-all bg-surface p-2 rounded border border-surface-border">{fileHash}</code>
              <p className="text-muted">
                Caso possua o código de verificação impresso no cabeçalho ou rodapé do documento, acerte o código na aba &quot;Digitar Código&quot; para conferir o selo deste arquivo contra a base do Projeto-nova-v2.
              </p>
            </div>
          )}
        </div>
      )}

      {/* Caixa de Esclarecimento Legal Institucional */}
      <div className="mt-4 flex flex-col gap-3 rounded-xl border border-surface-border/70 bg-surface-subtle/50 p-4 text-xs text-muted">
        <div className="flex items-center gap-2 font-medium text-foreground">
          <Shield size={16} className="text-primary" aria-hidden="true" />
          <span>Validade Jurídica & Integridade</span>
        </div>
        <p className="leading-relaxed">
          As assinaturas e carimbos de tempo conferidos por este portal estão em plena conformidade com a <strong>Lei Federal nº 14.063/2020</strong> (Assinatura Eletrônica Avançada na Administração Pública) e <strong>Medida Provisória nº 2.200-2/2001</strong>.
        </p>
        <div className="flex flex-wrap gap-4 pt-1 text-[11px] text-muted border-t border-surface-border/50">
          <span className="inline-flex items-center gap-1">
            <BookOpen size={13} aria-hidden="true" /> Resolução de Auditoria Transacional Imutável
          </span>
          <span className="inline-flex items-center gap-1">
            <ExternalLink size={13} aria-hidden="true" /> Carimbo de Tempo com Hora Legal Brasileira (HLB)
          </span>
        </div>
      </div>
    </div>
  );
}
