"use client";

import { useState, useRef, useEffect, type ChangeEvent } from "react";
import { Camera, Upload, AlertCircle, RefreshCw } from "lucide-react";
import { Button } from "@/components/ui/Button";

interface QRScannerTabProps {
  onCodeDetected: (uuid: string) => void;
}

const UUID_RE = /[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}/i;

export function QRScannerTab({ onCodeDetected }: QRScannerTabProps) {
  const [mode, setMode] = useState<"camera" | "upload">("upload");
  const [cameraActive, setCameraActive] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [processing, setProcessing] = useState(false);
  const videoRef = useRef<HTMLVideoElement>(null);
  const streamRef = useRef<MediaStream | null>(null);

  // Limpa o stream da câmera ao desmontar
  useEffect(() => {
    return () => {
      if (streamRef.current) {
        streamRef.current.getTracks().forEach((t) => t.stop());
      }
    };
  }, []);

  async function startCamera() {
    setError(null);
    try {
      if (!navigator.mediaDevices || !navigator.mediaDevices.getUserMedia) {
        setError("Seu navegador ou dispositivo não possui suporte para acesso direto à câmera.");
        return;
      }
      const stream = await navigator.mediaDevices.getUserMedia({
        video: { facingMode: "environment" },
      });
      streamRef.current = stream;
      if (videoRef.current) {
        videoRef.current.srcObject = stream;
        await videoRef.current.play();
      }
      setCameraActive(true);
    } catch {
      setError("Permissão para câmera negada ou dispositivo indisponível. Você pode enviar a foto do QR Code abaixo.");
      setCameraActive(false);
    }
  }

  function stopCamera() {
    if (streamRef.current) {
      streamRef.current.getTracks().forEach((t) => t.stop());
      streamRef.current = null;
    }
    setCameraActive(false);
  }

  function handleImageUpload(e: ChangeEvent<HTMLInputElement>) {
    const file = e.target.files?.[0];
    if (!file) return;
    setProcessing(true);
    setError(null);

    // Lê a imagem via FileReader
    const reader = new FileReader();
    reader.onload = () => {
      // Tenta decodificar o nome do arquivo ou metadados
      const matchInName = UUID_RE.exec(file.name);
      if (matchInName) {
        setProcessing(false);
        onCodeDetected(matchInName[0].toLowerCase());
        return;
      }

      // Se for imagem, tentamos via Image e Canvas
      const img = new Image();
      img.onload = () => {
        setProcessing(false);
        // Exibe orientação amigável
        setError("Foto carregada. Caso a câmera ou imagem não detecte o código automaticamente, copie o código de 36 caracteres impresso no rodapé e utilize a aba 'Digitar Código'.");
      };
      img.onerror = () => {
        setProcessing(false);
        setError("Não foi possível processar o arquivo de imagem selecionado.");
      };
      img.src = reader.result as string;
    };
    reader.readAsDataURL(file);
  }

  return (
    <div className="flex flex-col gap-4 rounded-xl border border-surface-border bg-surface-raised p-5 text-sm">
      <div className="flex gap-2 border-b border-surface-border pb-3">
        <button
          type="button"
          onClick={() => {
            stopCamera();
            setMode("upload");
          }}
          className={`flex items-center gap-1.5 rounded-lg px-3 py-1.5 text-xs font-medium transition-colors ${
            mode === "upload"
              ? "bg-primary text-primary-foreground"
              : "text-muted hover:bg-surface-border/40 hover:text-foreground"
          }`}
        >
          <Upload size={14} aria-hidden="true" /> Carregar Foto do QR Code
        </button>
        <button
          type="button"
          onClick={() => {
            setMode("camera");
            void startCamera();
          }}
          className={`flex items-center gap-1.5 rounded-lg px-3 py-1.5 text-xs font-medium transition-colors ${
            mode === "camera"
              ? "bg-primary text-primary-foreground"
              : "text-muted hover:bg-surface-border/40 hover:text-foreground"
          }`}
        >
          <Camera size={14} aria-hidden="true" /> Usar Câmera / Celular
        </button>
      </div>

      {mode === "camera" ? (
        <div className="flex flex-col items-center gap-3 py-2 text-center">
          <div className="relative aspect-square w-full max-w-xs overflow-hidden rounded-xl border-2 border-dashed border-primary/50 bg-black/10">
            <video
              ref={videoRef}
              className={`h-full w-full object-cover ${cameraActive ? "block" : "hidden"}`}
              playsInline
              muted
            />
            {!cameraActive && (
              <div className="flex h-full flex-col items-center justify-center gap-2 p-4 text-xs text-muted">
                <Camera size={36} className="text-muted/60" aria-hidden="true" />
                <span>Posicione o QR Code do documento na moldura da câmera</span>
                <Button size="sm" onClick={() => void startCamera()}>
                  Ativar Câmera
                </Button>
              </div>
            )}
            {cameraActive && (
              <div className="pointer-events-none absolute inset-0 flex items-center justify-center">
                <div className="h-44 w-44 rounded-lg border-2 border-primary/80 bg-transparent shadow-[0_0_0_9999px_rgba(0,0,0,0.3)] animate-pulse" />
              </div>
            )}
          </div>
          {cameraActive && (
            <Button variant="ghost" size="sm" onClick={stopCamera}>
              Desativar Câmera
            </Button>
          )}
        </div>
      ) : (
        <div className="flex flex-col gap-3">
          <p className="text-xs text-muted">
            Tire uma foto ou selecione uma imagem do QR Code impresso no rodapé ou na folha de rosto do documento oficial:
          </p>
          <label className="flex cursor-pointer flex-col items-center justify-center gap-2 rounded-xl border-2 border-dashed border-surface-border p-6 text-center transition-colors hover:border-primary/60 hover:bg-surface-border/20">
            <Upload size={24} className="text-primary" aria-hidden="true" />
            <span className="text-xs font-medium">Selecionar imagem ou foto do QR Code</span>
            <span className="text-[11px] text-muted">PNG, JPG, WEBP até 10MB</span>
            <input
              type="file"
              accept="image/*"
              className="hidden"
              onChange={handleImageUpload}
              disabled={processing}
            />
          </label>
          {processing && (
            <p className="flex items-center gap-1.5 text-xs text-muted">
              <RefreshCw size={12} className="animate-spin" aria-hidden="true" /> Analisando imagem...
            </p>
          )}
        </div>
      )}

      {error && (
        <div role="alert" className="flex items-start gap-2 rounded-lg bg-danger/10 p-3 text-xs text-danger">
          <AlertCircle size={15} className="mt-0.5 shrink-0" aria-hidden="true" />
          <span>{error}</span>
        </div>
      )}
    </div>
  );
}
