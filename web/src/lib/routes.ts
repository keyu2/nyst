const bulletFiles: Record<string, string> = {
  FX: "Fd",
  "6X": "6d",
  "7X": "7d",
  GS: "S",
  FS: "SF",
  H: "SR",
  SI: "SIR",
  SIX: "SIRd",
};

export const routeAliases: Record<string, string> = { S: "GS", SR: "H", SF: "FS", SIR: "SI", SS: "SIX" };

export const routeColors: Record<string, string> = {
  "1": "EE352E", "2": "EE352E", "3": "EE352E",
  "4": "00933C", "5": "00933C", "6": "00933C", "6X": "00933C",
  "7": "B933AD", "7X": "B933AD", A: "0039A6", C: "0039A6", E: "0039A6",
  B: "FF6319", D: "FF6319", F: "FF6319", FX: "FF6319", M: "FF6319",
  G: "6CBE45", J: "996633", Z: "996633", L: "A7A9AC",
  N: "FCCC0A", Q: "FCCC0A", R: "FCCC0A", W: "FCCC0A",
  GS: "808183", FS: "808183", H: "808183", SI: "0039A6", SIX: "0039A6",
};

export function getBullet(routeId: string) {
  return `/bullets/${bulletFiles[routeId] || routeId}.svg`;
}

export function normalizeRouteId(routeId: string) {
  return routeAliases[routeId] || routeId;
}

export function routeFamily(routeId: string) {
  const normalized = normalizeRouteId(routeId);
  return ({ "6X": "6", "7X": "7", FX: "F", SIX: "SI" } as Record<string, string>)[normalized] || normalized;
}

export function hasTransferRouteChange(currentRoutes: string[], previousRoutes: string[] = [], nextRoutes: string[] = []) {
  const previous = new Set(previousRoutes);
  const next = new Set(nextRoutes);
  return currentRoutes.some((routeId) => !previous.has(routeId) || !next.has(routeId));
}
