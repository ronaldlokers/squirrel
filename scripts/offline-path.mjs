// Does a chore typed with no network come back a chore?
//
// The one claim in this product that no Go test can make. It needs a real
// origin, a real service worker, and a real failure — and the worker is the
// only part of the screen that Go never runs.
//
//     node scripts/offline-path.mjs
//
// Needs chromium on the path and a checkout that builds. It starts and stops
// its own dev screen; nothing else has to be running.
//
// It stops the server rather than throttling the browser, and that is not
// belt and braces. CDP's Network.emulateNetworkConditions with offline:true
// does not reach the service worker's own fetch: the POST returned 200, the
// worker held nothing, and the run looked like a pass while proving only that
// the online path works. Killing the process is the only cut that reaches
// through.
//
// What a pass looks like, from 29 August 2026:
//
//     3. type a chore and submit, with nothing listening
//        action=/chores/name status=503 landed=/r/chores?held=1
//     4. what the worker is holding
//        [{"text":"descale the kettle","room":"chores",
//          "action":"/chores/name","field":"name"}]
//     6. what came back
//        chores: ["descale the kettle","How often should it come back? ..."]
//        in the pile: 0
//        still held: 0
//
// The two numbers at the end are the whole test. `in the pile: 0` is the
// failure this was written for — before the room travelled with the held note,
// every replay went to /capture as `text` and a chore came back a pile note.
// `still held: 0` is the queue draining rather than keeping what it delivered.
import { spawn, execSync } from 'node:child_process';
import { setTimeout as sleep } from 'node:timers/promises';

const REPO = '/home/ronald/Projects/github.com/ronaldlokers/squirrel';
const ADDR = '127.0.0.1:8426';
const BASE = `http://${ADDR}`;
const PORT = 9224;

let server = null;
const up = async () => {
  server = spawn('go', ['run', '-tags=dev', './cmd/devscreen', '--addr', ADDR],
    { cwd: REPO, stdio: 'ignore', detached: true });
  for (let i = 0; i < 90; i++) {
    try { const r = await fetch(`${BASE}/r/chores`); if (r.ok) return; } catch {}
    await sleep(500);
  }
  throw new Error('server never came up');
};
const down = async () => {
  if (server) { try { process.kill(-server.pid, 'SIGKILL'); } catch {} }
  try { execSync(`pkill -f 'devscreen --addr ${ADDR}'`); } catch {}
  for (let i = 0; i < 40; i++) {
    try { await fetch(`${BASE}/r/chores`); } catch { return; }
    await sleep(250);
  }
  throw new Error('server would not go down');
};

const chrome = spawn('chromium', ['--headless=new', `--remote-debugging-port=${PORT}`,
  '--disable-gpu', '--no-first-run',
  '--user-data-dir=/tmp/claude-1000/-home-ronald-Projects-github-com-ronaldlokers-squirrel/247265ee-d2fd-4c2c-a117-d087eaeea751/scratchpad/cdp2', 'about:blank'],
  { stdio: 'ignore' });
const bye = async () => { chrome.kill(); await down(); };
process.on('exit', () => { chrome.kill(); });

await up();

let target;
for (let i = 0; i < 40 && !target; i++) {
  try { target = (await (await fetch(`http://127.0.0.1:${PORT}/json/list`)).json()).find(t => t.type === 'page'); } catch {}
  if (!target) await sleep(250);
}
const ws = new WebSocket(target.webSocketDebuggerUrl);
await new Promise(r => ws.addEventListener('open', r));
let id = 0; const waiting = new Map();
ws.addEventListener('message', e => { const m = JSON.parse(e.data);
  if (m.id && waiting.has(m.id)) { waiting.get(m.id)(m); waiting.delete(m.id); } });
const send = (method, params = {}) => new Promise((res, rej) => {
  const n = ++id; waiting.set(n, m => m.error ? rej(new Error(m.error.message)) : res(m.result));
  ws.send(JSON.stringify({ id: n, method, params })); });
const run = async expr => {
  const r = await send('Runtime.evaluate', { expression: expr, awaitPromise: true, returnByValue: true });
  if (r.exceptionDetails) return 'THREW: ' + (r.exceptionDetails.exception?.description || '');
  return r.result.value; };

await send('Page.enable'); await send('Runtime.enable');

console.log('1. open the chores, register the worker');
await send('Page.navigate', { url: `${BASE}/r/chores` });
await sleep(3000);
console.log('   worker:', await run(`navigator.serviceWorker.getRegistration().then(r=>r&&r.active?'active':'not active')`));

console.log('2. STOP THE SERVER');
await down();
console.log('   server reachable:', await run(`fetch('${BASE}/r/chores').then(()=>'yes').catch(()=>'no')`));

console.log('3. type a chore and submit, with nothing listening');
console.log('   ', await run(`(async () => {
  const form = document.querySelector('.dock form.slot');
  const body = new URLSearchParams();
  new FormData(form).forEach((v,k) => body.append(k,v));
  body.set(form.querySelector('textarea').name, 'descale the kettle');
  try {
    const res = await fetch(form.action, {method:'POST', body,
      headers:{'Content-Type':'application/x-www-form-urlencoded'}, credentials:'same-origin'});
    return 'action=' + form.getAttribute('action') + ' status=' + res.status + ' landed=' + res.url;
  } catch (e) { return 'action=' + form.getAttribute('action') + ' fetch threw: ' + e.message; }
})()`));

console.log('4. what the worker is holding');
console.log('   ', await run(`(async () => {
  const db = await new Promise((res,rej)=>{const q=indexedDB.open('squirrel-held',1);
    q.onsuccess=()=>res(q.result); q.onerror=()=>rej(q.error); q.onupgradeneeded=()=>{};});
  if (![...db.objectStoreNames].includes('notes')) return 'no store';
  const st = db.transaction('notes').objectStore('notes');
  return await new Promise(res=>{const out=[];const c=st.openCursor();
    c.onsuccess=()=>{const cur=c.result; if(!cur) return res(JSON.stringify(out)); out.push(cur.value); cur.continue();};});
})()`));

console.log('5. START THE SERVER, then flush');
await up();
await run(`navigator.serviceWorker.ready.then(r=>r.active.postMessage('flush'))`);
await sleep(3000);

console.log('6. what came back');
await send('Page.navigate', { url: `${BASE}/r/chores` });
await sleep(2000);
console.log('   chores:', await run(`(()=>{const t=[...document.querySelectorAll('#thread .turn')]
  .map(e=>e.textContent.replace(/\\s+/g,' ').trim()); return JSON.stringify(t.slice(-2));})()`));
console.log('   in the pile:', await run(`fetch('${BASE}/r/pile').then(r=>r.text()).then(h=>(h.match(/descale the kettle/g)||[]).length)`));
console.log('   still held:', await run(`(async () => {
  const db = await new Promise(res=>{const q=indexedDB.open('squirrel-held',1);q.onsuccess=()=>res(q.result);q.onupgradeneeded=()=>{};});
  if (![...db.objectStoreNames].includes('notes')) return '0';
  const st=db.transaction('notes').objectStore('notes');
  return await new Promise(res=>{const c=st.count(); c.onsuccess=()=>res(String(c.result));});
})()`));

ws.close(); await bye();
