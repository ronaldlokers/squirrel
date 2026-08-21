// The service worker, and the smallest one that is worth having.
//
// It caches the things that cannot go stale — the stylesheet, the script, the
// fonts, the mark — and it does not cache the pile. A cached pile would show
// notes that have already been triaged, which is the two views disagreeing
// with each other, and it would do it in the one place you could not tell.
//
// So: assets from the cache when they are there, everything else straight to
// the network, and when the network is gone, a page that says so honestly. The
// note is already durable; this is a reader, not a queue.
const VERSION = "SQUIRREL_ASSET_VERSION";
const CACHE = `squirrel-${VERSION}`;

// Asset URLs carry ?v=<stamp>, so a new release asks for URLs this cache does
// not have and fetches them. The old cache is dropped on activate rather than
// updated in place — there is nothing in it worth keeping once its stamp is
// gone.
self.addEventListener("install", () => self.skipWaiting());

self.addEventListener("activate", event => {
  event.waitUntil((async () => {
    for (const name of await caches.keys()) {
      if (name !== CACHE) await caches.delete(name);
    }
    await self.clients.claim();
  })());
});

const OFFLINE = `<!doctype html><html lang="en"><head><meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>Squirrel</title></head>
<body style="background:#58388a;color:#fffbf3;font:16px/1.5 system-ui;padding:3rem">
<p>Squirrel cannot reach its memory from here. Nothing has been lost &mdash;
everything you said is still there, and the pile will be too when you are back
on a network.</p>
<p style="opacity:.75">Notes are still kept by talking to Squirrel in Campfire.</p>
</body></html>`;

// Words typed with no network, waiting for one.
//
// The screen writes straight to the pile and there is no spool behind that —
// the chat's 👀 means the words reached disk before anything else could go
// wrong, and a direct write has no such stage. This is the nearest honest
// substitute: the worker takes the words, the page says so, and they go in
// when the network comes back.
//
// IndexedDB rather than the cache, because these are not responses. Kept in
// the order they were said, because that is the only order that means
// anything, and deleted the moment they land — a queue that keeps what it has
// already delivered is a second pile.
const HELD = "squirrel-held";

function heldStore(mode) {
  return new Promise((resolve, reject) => {
    const open = indexedDB.open(HELD, 1);
    open.onupgradeneeded = () => open.result.createObjectStore("notes", { autoIncrement: true });
    open.onerror = () => reject(open.error);
    open.onsuccess = () => resolve(open.result.transaction("notes", mode).objectStore("notes"));
  });
}

function hold(text) {
  return heldStore("readwrite").then(store => new Promise((resolve, reject) => {
    const put = store.add({ text, at: Date.now() });
    put.onsuccess = () => resolve();
    put.onerror = () => reject(put.error);
  }));
}

// Send what is held, oldest first, and stop at the first failure — the network
// is either there or it is not, and hammering it with the rest achieves
// nothing. Each note is deleted only once its own write has landed.
async function flush() {
  const store = await heldStore("readwrite");
  const all = await new Promise((resolve, reject) => {
    const req = store.openCursor();
    const out = [];
    req.onsuccess = () => {
      const cursor = req.result;
      if (!cursor) return resolve(out);
      out.push({ key: cursor.key, ...cursor.value });
      cursor.continue();
    };
    req.onerror = () => reject(req.error);
  });

  for (const note of all) {
    const body = new URLSearchParams({ text: note.text });
    let res;
    try {
      res = await fetch("/capture", {
        method: "POST",
        headers: { "Content-Type": "application/x-www-form-urlencoded" },
        body,
        credentials: "same-origin",
      });
    } catch {
      return false;
    }
    if (!res.ok && res.status !== 303) return false;
    const del = await heldStore("readwrite");
    await new Promise(resolve => {
      const req = del.delete(note.key);
      req.onsuccess = req.onerror = () => resolve();
    });
  }
  return true;
}

self.addEventListener("message", event => {
  if (event.data === "flush") event.waitUntil(flush());
});

self.addEventListener("fetch", event => {
  const request = event.request;

  // A capture with no network is held rather than lost. The page is told by
  // the redirect it gets back, which is the same shape the server's own
  // answers take — one path through the page's code, whoever answered.
  //
  // Words only. A capture carrying a photograph goes straight past this and
  // out to the network like any other request, and that is a correctness rule
  // rather than an optimisation: everything below can hold is `text`, so the
  // one thing this could do with a photograph is drop it. It did exactly that.
  // A photograph on its own has no text at all, so the branch that decides
  // there was nothing to keep fired and answered a 303 to "/" — which is a
  // page that jumps to the top and shows no note, from a capture that never
  // reached the server.
  //
  // The photograph's durability is not lost by staying out of here. pile.js
  // holds a chosen photograph in IndexedDB from the moment it is picked and
  // puts it back on the input when the page comes back, so a failed post
  // leaves both the words and the picture on the screen to try again.
  //
  // Multipart is the test rather than "has a file part", because reading the
  // body to find out would consume the very request being forwarded.
  if (request.method === "POST" && new URL(request.url).pathname === "/capture" &&
      !(request.headers.get("Content-Type") || "").startsWith("multipart/")) {
    event.respondWith((async () => {
      const copy = request.clone();
      try {
        return await fetch(request);
      } catch {
        const text = (await copy.formData()).get("text");
        if (!text || !String(text).trim()) return Response.redirect("/", 303);
        await hold(String(text));
        return Response.redirect("/?held=1", 303);
      }
    })());
    return;
  }

  if (request.method !== "GET") return;

  const url = new URL(request.url);
  const isAsset = url.pathname.includes("/static/");

  if (isAsset) {
    // Cache-first, because a stamped URL names one specific version of one
    // specific file: if it is in the cache it is the right bytes by
    // definition.
    event.respondWith((async () => {
      const hit = await caches.match(request);
      if (hit) return hit;
      const res = await fetch(request);
      if (res.ok) (await caches.open(CACHE)).put(request, res.clone());
      return res;
    })());
    return;
  }

  // Everything else is the pile, and the pile is state. Straight to the
  // network, and an honest page when there is none.
  event.respondWith(
    fetch(request).catch(() =>
      new Response(OFFLINE, { headers: { "Content-Type": "text/html; charset=utf-8" }, status: 503 })
    )
  );
});

// The one thing that arrives when you are not looking at the screen.
//
// Only the leave-by warning is ever pushed. A nudge is a suggestion and a
// suggestion that waits is doing its job; "leave about 14:10" has one useful
// minute, and a message noticed at 14:40 is worse than none because it teaches
// you not to trust the next one.
//
// requireInteraction is deliberate and is the only place this product asks for
// your attention rather than waiting for it: a notification that vanishes into
// the shade while you are in another room is the failure this exists to fix.
self.addEventListener("push", event => {
  event.waitUntil((async () => {
    let said = { title: "Squirrel", body: "" };
    try {
      if (event.data) said = { ...said, ...event.data.json() };
    } catch {
      // A push that will not parse is still worth showing: something is
      // happening and staying silent about it is the one outcome with no
      // recovery.
      said.body = event.data ? event.data.text() : "";
    }
    await self.registration.showNotification(said.title, {
      body: said.body,
      icon: "/static/icon-192.png",
      badge: "/static/icon-192.png",
      // One at a time. A stack of these is a list of things you are late for,
      // which is the shape this product does not have.
      tag: "squirrel-now",
      renotify: true,
      requireInteraction: true
    });
  })());
});

// Pressing it goes to the front door rather than to a deep link. The offer is
// there, and a link to something that has since been done is worse than a page
// that says what is true now.
self.addEventListener("notificationclick", event => {
  event.notification.close();
  event.waitUntil((async () => {
    const open = await self.clients.matchAll({ type: "window", includeUncontrolled: true });
    for (const client of open) {
      if (new URL(client.url).origin === self.location.origin) return client.focus();
    }
    return self.clients.openWindow("/");
  })());
});
