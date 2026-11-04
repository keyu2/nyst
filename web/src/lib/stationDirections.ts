import type { Arrival } from "./api";
import {routeFamily} from "./routes.ts";
import {directionGroupLabel,disambiguateDirectionGroupLabels} from "./directionLabels.ts";

export type DirectionOption={id:string;label:string;borough?:string;direction:string;trains:Arrival[]};
export type StationDirectionGroup={id:string;label:string;options:DirectionOption[]};
export function realtimeDirectionOptions({activeRoute,northbound,southbound}:{activeRoute:string;northbound:Arrival[];southbound:Arrival[]}) {
  const result:DirectionOption[]=[];
  for(const [direction,trains] of [["northbound",northbound],["southbound",southbound]] as const){
    const destinations=new Map<string,DirectionOption>();
    for(const train of trains){
      if(routeFamily(train.trip.route.id)!==activeRoute)continue;
      const destinationId=train.destination?.id||train.destination?.name||train.headsign;
      const label=train.destination?.name||train.headsign;
      if(!destinationId||!label)continue;
      const key=`${direction}:${destinationId}`;
      const option=destinations.get(key)||{id:key,label,borough:train.destination?.borough,direction,trains:[]};
      option.trains.push(train);destinations.set(key,option);
    }
    result.push(...destinations.values());
  }
  return result;
}
export function directionKey(option:DirectionOption){return `${option.id.split(":",1)[0]}:${option.label.trim().toLocaleLowerCase()}`;}
export function groupDirectionOptions(options:DirectionOption[],borough?:string){
  const groups:StationDirectionGroup[]=[];
  for(const option of options){
    const label=directionGroupLabel(borough,option.borough,option.direction);
    const group=groups.find((item)=>item.id===option.direction);
    if(!group){groups.push({id:option.direction,label,options:[option]});continue;}
    group.options.push(option);
    if(label.includes("/")||!group.label.includes("/"))group.label=label;
  }
  return disambiguateDirectionGroupLabels(groups);
}
