import {spawn} from "node:child_process";
import {once} from "node:events";
import {watch} from "node:fs";
import {access,mkdtemp,rm} from "node:fs/promises";
import {tmpdir} from "node:os";
import {createServer as createPortProbe} from "node:net";
import {join} from "node:path";
import {fileURLToPath} from "node:url";
import {build,createServer} from "vite";

const root=fileURLToPath(new URL("../",import.meta.url));
const web=fileURLToPath(new URL("./",import.meta.url));
const temporary=await mkdtemp(join(tmpdir(),"nyst-dev-"));
const binary=join(temporary,process.platform==="win32"?"nyst.exe":"nyst");
let apiPort;
let backend;
let compiler;
let stopping=false;
let building=false;
let requested=false;
let timer;
const watchers=[];
let vite;

async function availablePort(port){
  const probe=createPortProbe();
  await new Promise((resolve,reject)=>{probe.once("error",reject);probe.listen(port,resolve)});
  const selected=probe.address().port;
  await new Promise((resolve,reject)=>probe.close((error)=>error?reject(error):resolve()));
  return selected;
}

async function stopProcess(child){
  if(!child||child.exitCode!==null||child.signalCode)return;
  const ended=once(child,"exit");
  const timeout=setTimeout(()=>child.kill("SIGKILL"),3000);
  child.kill("SIGTERM");
  await ended;
  clearTimeout(timeout);
}

async function waitForBackend(child){
  const deadline=Date.now()+10000;
  while(!stopping&&Date.now()<deadline&&child.exitCode===null&&child.signalCode===null){
    try{
      await fetch(`http://127.0.0.1:${apiPort}/api/health`,{signal:AbortSignal.timeout(1000)});
      return true;
    }catch{}
    await new Promise((resolve)=>setTimeout(resolve,50));
  }
  return false;
}

async function rebuild(){
  requested=true;
  if(building||stopping)return;
  building=true;
  try{
    while(requested&&!stopping){
      requested=false;
      console.log("[Go] Building backend…");
      compiler=spawn("go",["build","-o",binary,"."],{cwd:root,stdio:"inherit"});
      const [code]=await once(compiler,"exit");
      compiler=undefined;
      if(stopping)break;
      if(code!==0){console.error("[Go] Build failed. Save a correction to retry.");continue;}
      await stopProcess(backend);
      if(stopping)break;
      backend=spawn(binary,[],{cwd:root,stdio:"inherit",env:{...process.env,PORT:apiPort}});
      backend.on("error",(error)=>console.error("[Go]",error.message));
      backend.on("exit",(code)=>{if(code&&!stopping)console.error(`[Go] Backend exited (${code}).`)});
      if(!await waitForBackend(backend))continue;
      console.log(`[Go] Backend ready on port ${apiPort}`);
      vite?.ws.send({type:"full-reload",path:"*"});
    }
  }finally{building=false;}
}

async function close(){
  if(stopping)return;
  stopping=true;
  clearTimeout(timer);
  watchers.forEach((watcher)=>watcher.close());
  await Promise.allSettled([stopProcess(compiler),stopProcess(backend),vite?.close()]);
  await rm(temporary,{recursive:true,force:true});
}

try{
  const webPort=Number(process.env.NYST_PORT??5173);
  const backendPort=Number(process.env.NYST_API_PORT??webPort+1);
  if(!Number.isInteger(webPort)||webPort<1||webPort>65534)throw new Error("NYST_PORT must be an integer from 1 to 65534.");
  if(!Number.isInteger(backendPort)||backendPort<1||backendPort>65535||backendPort===webPort)throw new Error("NYST_API_PORT must be a different port from 1 to 65535.");
  await availablePort(webPort);
  apiPort=await availablePort(backendPort);
  try{await access(join(web,"dist/index.html"));}catch{await build({root:web});}
  vite=await createServer({root:web,server:{port:webPort,strictPort:true,proxy:{"/api":`http://127.0.0.1:${apiPort}`}}});
  await vite.listen();
  vite.printUrls();
  const changed=(_event,file)=>{
    if(!file||!/(?:\.go|go\.mod|go\.sum)$/.test(String(file)))return;
    clearTimeout(timer);
    timer=setTimeout(()=>{void rebuild().catch((error)=>console.error("[Go]",error.message))},150);
  };
  watchers.push(watch(join(root,"server"),{recursive:true},changed));
  watchers.push(watch(root,changed));
  process.on("SIGINT",()=>{void close()});
  process.on("SIGTERM",()=>{void close()});
  await rebuild();
}catch(error){
  console.error(error.message);
  await close();
  process.exitCode=1;
}
