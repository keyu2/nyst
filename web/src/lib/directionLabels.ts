const boroughLabels: Record<string, string> = {
  Bronx: "The Bronx",
  Brooklyn: "Brooklyn",
  Manhattan: "Manhattan",
  Queens: "Queens",
  "Staten Island": "Staten Island",
};

export function directionGroupLabel(originBorough: string | undefined, destinationBorough: string | undefined, direction: string) {
  const northbound = direction === "northbound";
  if (originBorough === "Manhattan") {
    if (northbound) return destinationBorough === "Bronx" ? "Uptown / The Bronx" : "Uptown";
    return destinationBorough === "Brooklyn" ? "Downtown / Brooklyn" : "Downtown";
  }
  if (originBorough === "Bronx") {
    if (destinationBorough === "Bronx") return "The Bronx";
    if (destinationBorough === "Brooklyn") return "Manhattan / Brooklyn";
    return boroughLabels[destinationBorough || ""] || (northbound ? "The Bronx" : "Manhattan");
  }
  if (originBorough === "Brooklyn") {
    if (destinationBorough === "Brooklyn") return "Brooklyn";
    if (destinationBorough === "Bronx") return "Manhattan / The Bronx";
    return boroughLabels[destinationBorough || ""] || (northbound ? "Manhattan" : "Brooklyn");
  }
  if (originBorough === "Queens") return boroughLabels[destinationBorough || ""] || (northbound ? "Queens" : "Manhattan");
  if (originBorough === "Staten Island") return boroughLabels[destinationBorough || ""] || "Staten Island";
  return boroughLabels[destinationBorough || ""] || (northbound ? "Uptown" : "Downtown");
}

export type DirectionGroup = { label: string; options?: { label?: string }[] };

export function disambiguateDirectionGroupLabels<T extends DirectionGroup>(groups: T[]): T[] {
  const labelCounts = new Map<string, number>();
  for (const group of groups) labelCounts.set(group.label, (labelCounts.get(group.label) || 0) + 1);
  return groups.map((group) => {
    if ((labelCounts.get(group.label) || 0) < 2) return group;
    const terminalNames = [...new Set((group.options || []).map((option) => option.label).filter(Boolean))];
    return { ...group, label: terminalNames.join(" / ") || group.label };
  });
}

export function directionLabel(origin: string | undefined, trains: { destination: { borough?: string } }[], direction: string) {
  const groups = trains.map((train) => ({
    label: directionGroupLabel(origin, train.destination.borough, direction),
    options: [{ label: "" }],
  }));
  const labels = [...new Set(disambiguateDirectionGroupLabels(groups).map((group) => group.label))];
  return labels.join(" / ") || directionGroupLabel(origin, undefined, direction);
}
