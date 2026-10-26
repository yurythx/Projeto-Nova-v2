/**
 * Gerador de QR Code puramente vetorial SVG em TypeScript (Zero dependências).
 * Compatível com TypeScript estrito (noUncheckedIndexedAccess).
 */

const RS_FACTORS: Record<number, number[]> = {
  7: [0, 87, 229, 146, 149, 238, 102, 21],
  10: [0, 251, 67, 46, 61, 118, 70, 64, 94, 32, 45],
  15: [0, 29, 196, 111, 163, 112, 74, 10, 105, 105, 139, 132, 151, 32, 134, 26],
  18: [0, 215, 234, 158, 94, 184, 97, 120, 170, 214, 175, 40, 211, 106, 92, 244, 213, 207, 7],
  20: [
    0, 17, 60, 79, 50, 61, 163, 26, 187, 202, 180, 221, 225, 83, 239, 156, 164, 212, 212, 188, 190,
  ],
  24: [
    0, 117, 144, 219, 168, 90, 158, 226, 71, 76, 54, 36, 122, 129, 150, 194, 205, 207, 158, 236,
    220, 42, 173, 219, 211,
  ],
  28: [
    0, 183, 26, 201, 87, 210, 221, 113, 21, 46, 65, 45, 50, 238, 184, 249, 225, 102, 58, 209, 218,
    109, 165, 26, 95, 184, 192, 181, 158,
  ],
  30: [
    0, 212, 246, 77, 73, 195, 89, 42, 86, 172, 223, 211, 9, 213, 218, 19, 18, 247, 196, 168, 247,
    178, 143, 211, 134, 85, 84, 221, 60, 58, 77,
  ],
};

const EXP_TABLE = new Uint8Array(512);
const LOG_TABLE = new Uint8Array(256);
(() => {
  let x = 1;
  for (let i = 0; i < 255; i++) {
    EXP_TABLE[i] = x;
    EXP_TABLE[i + 255] = x;
    LOG_TABLE[x] = i;
    x <<= 1;
    if (x >= 256) x ^= 0x11d;
  }
})();

function gMult(a: number, b: number): number {
  if (a === 0 || b === 0) return 0;
  const logA = LOG_TABLE[a] ?? 0;
  const logB = LOG_TABLE[b] ?? 0;
  return EXP_TABLE[logA + logB] ?? 0;
}

interface VersionInfo {
  version: number;
  size: number;
  dataBytesL: number;
  ecBytesL: number;
  dataBytesM: number;
  ecBytesM: number;
  alignments: number[];
}

const VERSIONS: VersionInfo[] = [
  {
    version: 1,
    size: 21,
    dataBytesL: 19,
    ecBytesL: 7,
    dataBytesM: 16,
    ecBytesM: 10,
    alignments: [],
  },
  {
    version: 2,
    size: 25,
    dataBytesL: 34,
    ecBytesL: 10,
    dataBytesM: 28,
    ecBytesM: 16,
    alignments: [6, 18],
  },
  {
    version: 3,
    size: 29,
    dataBytesL: 55,
    ecBytesL: 15,
    dataBytesM: 44,
    ecBytesM: 26,
    alignments: [6, 22],
  },
  {
    version: 4,
    size: 33,
    dataBytesL: 80,
    ecBytesL: 20,
    dataBytesM: 64,
    ecBytesM: 36,
    alignments: [6, 26],
  },
  {
    version: 5,
    size: 37,
    dataBytesL: 108,
    ecBytesL: 26,
    dataBytesM: 86,
    ecBytesM: 48,
    alignments: [6, 30],
  },
  {
    version: 6,
    size: 41,
    dataBytesL: 136,
    ecBytesL: 36,
    dataBytesM: 108,
    ecBytesM: 64,
    alignments: [6, 34],
  },
];

export interface QRCodeData {
  size: number;
  modules: boolean[][];
}

export function generateQRCode(text: string, level: "L" | "M" = "M"): QRCodeData {
  const encoder = new TextEncoder();
  const rawBytes = encoder.encode(text);
  const dataLen = rawBytes.length;

  let target: VersionInfo = VERSIONS[VERSIONS.length - 1]!;
  for (const v of VERSIONS) {
    const maxCapacity = level === "L" ? v.dataBytesL - 3 : v.dataBytesM - 3;
    if (dataLen <= maxCapacity) {
      target = v;
      break;
    }
  }

  const numDataBytes = level === "L" ? target.dataBytesL : target.dataBytesM;
  const numEcBytes = level === "L" ? target.ecBytesL : target.ecBytesM;

  const bits: number[] = [];
  const pushBits = (val: number, len: number) => {
    for (let i = len - 1; i >= 0; i--) bits.push((val >> i) & 1);
  };

  pushBits(0b0100, 4);
  pushBits(dataLen, 8);
  for (const b of rawBytes) pushBits(b, 8);

  const maxBits = numDataBytes * 8;
  const termLen = Math.min(4, maxBits - bits.length);
  pushBits(0, termLen);

  while (bits.length % 8 !== 0) bits.push(0);

  const pad = [0xec, 0x11];
  let padIdx = 0;
  while (bits.length < maxBits) {
    const padVal = pad[padIdx % 2] ?? 0xec;
    pushBits(padVal, 8);
    padIdx++;
  }

  const dataBytes = new Uint8Array(numDataBytes);
  for (let i = 0; i < numDataBytes; i++) {
    let b = 0;
    for (let j = 0; j < 8; j++) {
      const bit = bits[i * 8 + j] ?? 0;
      b = (b << 1) | bit;
    }
    dataBytes[i] = b;
  }

  const defaultFactors = [0, 251, 67, 46, 61, 118, 70, 64, 94, 32, 45];
  const factors = RS_FACTORS[numEcBytes] ?? defaultFactors;
  const ec = new Uint8Array(numEcBytes);
  for (let i = 0; i < dataBytes.length; i++) {
    const dByte = dataBytes[i] ?? 0;
    const ecFirst = ec[0] ?? 0;
    const factor = dByte ^ ecFirst;
    for (let j = 0; j < numEcBytes - 1; j++) {
      const nextEc = ec[j + 1] ?? 0;
      const poly = factors[j + 1] ?? 0;
      ec[j] = nextEc ^ gMult(poly, factor);
    }
    const lastPoly = factors[numEcBytes] ?? 0;
    ec[numEcBytes - 1] = gMult(lastPoly, factor);
  }

  const allBytes = new Uint8Array(numDataBytes + numEcBytes);
  allBytes.set(dataBytes, 0);
  allBytes.set(ec, numDataBytes);

  const finalBits: number[] = [];
  for (const b of allBytes) {
    for (let i = 7; i >= 0; i--) finalBits.push((b >> i) & 1);
  }

  const size = target.size;
  const matrix: (boolean | null)[][] = Array.from({ length: size }, () => Array(size).fill(null));

  const setCell = (r: number, c: number, val: boolean) => {
    const row = matrix[r];
    if (row && c >= 0 && c < size) {
      row[c] = val;
    }
  };

  const getCell = (r: number, c: number): boolean | null => {
    const row = matrix[r];
    return row ? (row[c] ?? null) : null;
  };

  const addFinder = (row: number, col: number) => {
    for (let r = -1; r <= 7; r++) {
      for (let c = -1; c <= 7; c++) {
        const nr = row + r;
        const nc = col + c;
        if (nr >= 0 && nr < size && nc >= 0 && nc < size) {
          const isBorder = r === -1 || r === 7 || c === -1 || c === 7;
          if (isBorder) {
            setCell(nr, nc, false);
          } else {
            const isRing = r === 0 || r === 6 || c === 0 || c === 6;
            const isCenter = r >= 2 && r <= 4 && c >= 2 && c <= 4;
            setCell(nr, nc, isRing || isCenter);
          }
        }
      }
    }
  };
  addFinder(0, 0);
  addFinder(0, size - 7);
  addFinder(size - 7, 0);

  for (let i = 8; i < size - 8; i++) {
    if (getCell(6, i) === null) setCell(6, i, i % 2 === 0);
    if (getCell(i, 6) === null) setCell(i, 6, i % 2 === 0);
  }

  setCell(4 * target.version + 9, 8, true);

  if (target.alignments.length > 0) {
    for (const r of target.alignments) {
      for (const c of target.alignments) {
        if (getCell(r, c) !== null) continue;
        for (let dr = -2; dr <= 2; dr++) {
          for (let dc = -2; dc <= 2; dc++) {
            const isBorder = Math.abs(dr) === 2 || Math.abs(dc) === 2;
            const isCenter = dr === 0 && dc === 0;
            setCell(r + dr, c + dc, isBorder || isCenter);
          }
        }
      }
    }
  }

  for (let i = 0; i < 9; i++) {
    if (getCell(8, i) === null) setCell(8, i, false);
    if (getCell(i, 8) === null) setCell(i, 8, false);
  }
  for (let i = 0; i < 8; i++) {
    if (getCell(8, size - 1 - i) === null) setCell(8, size - 1 - i, false);
    if (getCell(size - 1 - i, 8) === null) setCell(size - 1 - i, 8, false);
  }

  let bitIdx = 0;
  let dir = -1;
  let col = size - 1;
  while (col > 0) {
    if (col === 6) col--;
    const rowRange =
      dir === -1
        ? Array.from({ length: size }, (_, i) => size - 1 - i)
        : Array.from({ length: size }, (_, i) => i);

    for (const r of rowRange) {
      for (const c of [col, col - 1]) {
        if (getCell(r, c) === null) {
          const bit = bitIdx < finalBits.length ? (finalBits[bitIdx++] ?? 0) : 0;
          const mask = (r + c) % 2 === 0;
          setCell(r, c, (bit === 1) !== mask);
        }
      }
    }
    dir = -dir;
    col -= 2;
  }

  const formatBits = [1, 0, 1, 0, 1, 0, 0, 0, 0, 0, 1, 0, 0, 1, 0];
  for (let i = 0; i < 6; i++) setCell(8, i, Boolean(formatBits[i]));
  setCell(8, 7, Boolean(formatBits[6]));
  setCell(8, 8, Boolean(formatBits[7]));
  setCell(7, 8, Boolean(formatBits[8]));
  for (let i = 9; i < 15; i++) setCell(14 - i, 8, Boolean(formatBits[i]));

  for (let i = 0; i < 8; i++) setCell(size - 1 - i, 8, Boolean(formatBits[i]));
  for (let i = 8; i < 15; i++) setCell(8, size - 15 + i, Boolean(formatBits[i]));

  return {
    size,
    modules: matrix.map((row) => row.map((cell) => Boolean(cell))),
  };
}
