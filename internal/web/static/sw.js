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

self.addEventListener("fetch", event => {
  const request = event.request;
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
