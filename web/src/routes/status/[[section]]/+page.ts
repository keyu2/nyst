import { error } from "@sveltejs/kit";
export function load({ params }: { params: { section?: string } }) {
  if (params.section && !["subway", "favorites"].includes(params.section)) error(404);
}
