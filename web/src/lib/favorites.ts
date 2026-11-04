export type FavoriteStation = { id:string; ids:string[]; name:string; route:string; routes:string[] };
const linesKey="subway-favorite-lines";
const stationsKey="subway-favorite-stations";
function read(key:string):unknown {try{return JSON.parse(localStorage.getItem(key)||"null")}catch{return null}}
function strings(value:unknown):string[] {return Array.isArray(value)?value.filter((item):item is string=>typeof item==="string"):[]}
function save(key:string,value:unknown){
  try{localStorage.setItem(key,JSON.stringify(value))}catch{}
  window.dispatchEvent(new CustomEvent(key+"-change"));
}
export function favoriteLines(){
  try{
    const cookie=document.cookie.split("; ").find((item)=>item.startsWith(linesKey+"="));
    if(cookie){const value=JSON.parse(decodeURIComponent(cookie.slice(linesKey.length+1)));if(Array.isArray(value))return [...new Set(strings(value))]}
  }catch{}
  return [...new Set(strings(read(linesKey)))];
}
export function toggleFavoriteLine(route:string){
  const current=favoriteLines();
  const next=current.includes(route)?current.filter((item)=>item!==route):[...current,route];
  document.cookie=`${linesKey}=${encodeURIComponent(JSON.stringify(next))}; Max-Age=31536000; Path=/; SameSite=Lax`;
  save(linesKey,next);
}
export function favoriteStations():FavoriteStation[]{
  const value=read(stationsKey);
  if(!Array.isArray(value))return [];
  return value.filter((item)=>item&&typeof item.id==="string"&&typeof item.name==="string").map((item)=>({
    id:item.id,ids:Array.isArray(item.ids)?[...new Set(strings(item.ids))]:[item.id],name:item.name,
    route:typeof item.route==="string"?item.route:"",routes:strings(item.routes),
  }));
}
function stationIDs(station:FavoriteStation){return station.ids.length?station.ids:[station.id]}
export function isFavoriteStation(ids:string[]){const targets=new Set(ids);return favoriteStations().some((station)=>stationIDs(station).some((id)=>targets.has(id)))}
export function toggleFavoriteStation(station:FavoriteStation){
  const current=favoriteStations();const targets=new Set(stationIDs(station));
  const exists=current.some((item)=>stationIDs(item).some((id)=>targets.has(id)));
  save(stationsKey,exists?current.filter((item)=>!stationIDs(item).some((id)=>targets.has(id))):[...current,station]);
  return !exists;
}
export function removeFavoriteStation(id:string){save(stationsKey,favoriteStations().filter((station)=>station.id!==id))}
export function watchFavorites(update: () => void) {
  const events = ["storage", linesKey+"-change", stationsKey+"-change"];
  events.forEach((event) => window.addEventListener(event, update));
  return () => events.forEach((event) => window.removeEventListener(event, update));
}
