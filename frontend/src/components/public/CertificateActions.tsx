"use client";

import { useState } from "react";
import { Printer, Copy, Check, Share2, QrCode } from "lucide-react";
import { Button } from "@/components/ui/Button";
import { QRCodeSVG } from "@/components/ui/QRCodeSVG";

interface CertificateActionsProps {
  envelopeId: string;
  documentSha256: string;
  title: string;
}

export function CertificateActions({ envelopeId }: CertificateActionsProps) {
  const [copiedCode, setCopiedCode] = useState(false);
  const [copiedLink, setCopiedLink] = useState(false);
  const [showQRModal, setShowQRModal] = useState(false);

  const verifyUrl = typeof window !== "undefined"
    ? `${window.location.origin}/verificar/${envelopeId}`
    : `https://nexus.rondonopolis.mt.gov.br/verificar/${envelopeId}`;

  function copyCode() {
    void navigator.clipboard.writeText(envelopeId);
    setCopiedCode(true);
    setTimeout(() => setCopiedCode(false), 2000);
  }

  function copyLink() {
    void navigator.clipboard.writeText(verifyUrl);
    setCopiedLink(true);
    setTimeout(() => setCopiedLink(false), 2000);
  }

  function handlePrint() {
    window.print();
  }

  return (
    <div className="flex flex-col gap-3">
      <div className="flex flex-wrap items-center gap-2 print:hidden">
        <Button variant="secondary" size="sm" onClick={handlePrint}>
          <Printer size={15} aria-hidden="true" className="mr-1.5" />
          Imprimir Certidão Oficial
        </Button>

        <Button variant="ghost" size="sm" onClick={copyCode}>
          {copiedCode ? (
            <Check size={15} className="mr-1.5 text-success" aria-hidden="true" />
          ) : (
            <Copy size={15} aria-hidden="true" className="mr-1.5" />
          )}
          {copiedCode ? "Código Copiado!" : "Copiar Código"}
        </Button>

        <Button variant="ghost" size="sm" onClick={copyLink}>
          {copiedLink ? (
            <Check size={15} className="mr-1.5 text-success" aria-hidden="true" />
          ) : (
            <Share2 size={15} aria-hidden="true" className="mr-1.5" />
          )}
          {copiedLink ? "Link Copiado!" : "Compartilhar Link"}
        </Button>

        <Button
          variant="ghost"
          size="sm"
          onClick={() => setShowQRModal(!showQRModal)}
          className="text-primary hover:text-primary"
        >
          <QrCode size={15} aria-hidden="true" className="mr-1.5" />
          {showQRModal ? "Ocultar QR Code" : "Ver QR Code"}
        </Button>
      </div>

      {/* QR Code Interativo em Tela */}
      {showQRModal && (
        <div className="flex flex-col items-center gap-3 rounded-xl border border-surface-border bg-surface-raised p-5 text-center animate-fadeIn print:hidden">
          <div className="rounded-xl border border-surface-border bg-white p-3 shadow-xs">
            <QRCodeSVG value={verifyUrl} size={160} fgColor="#0f172a" bgColor="#ffffff" />
          </div>
          <div className="max-w-sm text-xs text-muted">
            <p className="font-medium text-foreground">Aponte a câmera para conferir este documento</p>
            <p className="mt-1 break-all font-mono text-[11px] text-muted">{verifyUrl}</p>
          </div>
        </div>
      )}

      {/* Bloco exclusivo para Impressão da Certidão (@media print) */}
      <div className="hidden print:flex flex-row items-center justify-between border-t border-b border-surface-border py-4 mt-6">
        <div className="flex flex-col gap-1 text-xs">
          <p className="font-bold">PROJETO-NOVA-V2 · MICROKERNEL</p>
          <p>Validação Oficial via Lei Federal nº 14.063/2020 e MP nº 2.200-2/2001</p>
          <p className="font-mono text-[10px]">URL: {verifyUrl}</p>
        </div>
        <div className="bg-white p-1 rounded border border-black/20">
          <QRCodeSVG value={verifyUrl} size={90} fgColor="#000000" bgColor="#ffffff" />
        </div>
      </div>
    </div>
  );
}
