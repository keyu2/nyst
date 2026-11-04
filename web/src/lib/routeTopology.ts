import type { RouteBranch, RouteStop } from "./api";

function branchById(branches: RouteBranch[], branchId: string) {
  return branches.find((branch) => branch.id === branchId);
}

export function buildRouteTopology(stops: RouteStop[], branches: RouteBranch[], routeId: string) {
  if (routeId === "A") {
    const lefferts = branchById(branches, "lefferts");
    const farRockaway = branchById(branches, "far-rockaway");
    if (lefferts && farRockaway) {
      const rockawayPark = branchById(branches, "rockaway-park");
      const broadChannelIndex = farRockaway.stops.findIndex((stop) => stop.id === "H04");
      const farRockawayTrunk = broadChannelIndex >= 0 ? farRockaway.stops.slice(0, broadChannelIndex + 1) : farRockaway.stops;
      const farRockawayTail = broadChannelIndex >= 0 ? farRockaway.stops.slice(broadChannelIndex + 1) : [];
      return { kind: "a" as const, lefferts, rockawayPark, farRockawayTrunk, farRockawayTail, hasRockawayParkSplit: Boolean(rockawayPark && farRockawayTail.length) };
    }
  }
  if (routeId === "5") {
    const branch = branchById(branches, "nereid");
    const junctionIndex = stops.findIndex((stop) => stop.id === "213");
    if (branch && junctionIndex > 0) return { kind: "five" as const, trunkBefore: stops.slice(0, junctionIndex), branch, trunkAfter: stops.slice(junctionIndex) };
  }
  const temporaryBranches = branches.filter((branch) => branch.temporary && branch.junctionId);
  if (temporaryBranches.length) {
    const junctionIndex = stops.findIndex((stop) => stop.id === temporaryBranches[0].junctionId);
    if (junctionIndex >= 0) {
      const branchesBefore = temporaryBranches.filter((branch) => branch.placement === "before");
      const branchesAfter = temporaryBranches.filter((branch) => branch.placement !== "before");
      const splitIndex = junctionIndex + (branchesBefore.length ? 0 : 1);
      return { kind: "temporary" as const, trunkBefore: stops.slice(0, splitIndex), branchesBefore, branchesAfter, trunkAfter: stops.slice(splitIndex) };
    }
  }
  return { kind: "standard" as const, branches: branches.filter((branch) => !branch.temporary) };
}
