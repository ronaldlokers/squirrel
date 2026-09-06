(function () {
  (() => {
    const form = document.querySelector("form.blankstrip");
    if (!form) return;
    const input = typeof DataTransfer === "undefined"
      ? null : form.querySelector('input[name="photo"]');
    if (!input) return;

    const DB = "squirrel-photo", STORE = "photo", ONE = "pending";

    function open() {
      return new Promise((resolve, reject) => {
        const req = indexedDB.open(DB, 1);
        req.onupgradeneeded = () => req.result.createObjectStore(STORE);
        req.onerror = () => reject(req.error);
        req.onsuccess = () => resolve(req.result);
      });
    }

    function inStore(mode, run) {
      return open().then(db => new Promise((resolve, reject) => {
        const req = run(db.transaction(STORE, mode).objectStore(STORE));
        req.onsuccess = () => resolve(req.result);
        req.onerror = () => reject(req.error);
      }));
    }

    const stash = file => inStore("readwrite", s => s.put(file, ONE));
    const forget = () => inStore("readwrite", s => s.delete(ONE));
    const stashed = () => inStore("readonly", s => s.get(ONE));

    const shown = document.createElement("div");
    shown.className = "gotphoto";
    shown.hidden = true;
    const thumb = document.createElement("img");
    thumb.alt = "the photograph you are about to keep";
    const off = document.createElement("button");
    off.type = "button";
    off.className = "unphoto";
    off.textContent = "take it off";
    shown.append(thumb, off);
    form.append(shown);

    let drawn = "";

    function show(file) {
      if (drawn) URL.revokeObjectURL(drawn);
      drawn = URL.createObjectURL(file);
      thumb.src = drawn;
      shown.hidden = false;
    }

    function hide() {
      if (drawn) URL.revokeObjectURL(drawn);
      drawn = "";
      thumb.removeAttribute("src");
      shown.hidden = true;
    }

    function enctypeFor() {
      form.enctype = input.files?.length
        ? "multipart/form-data"
        : "application/x-www-form-urlencoded";
    }

    input.addEventListener("change", () => {
      const file = input.files?.[0];
      enctypeFor();
      if (!file) { hide(); forget().catch(() => {}); return; }
      show(file);
      stash(file).catch(() => {});
    });

    off.addEventListener("click", () => {
      input.value = "";
      enctypeFor();
      hide();
      forget().catch(() => {});
    });

    (async () => {
      try {
        if (new URLSearchParams(location.search).has("kept")) {
          await forget();
          return;
        }
        const file = await stashed();
        if (!file) return;
        const carrier = new DataTransfer();
        carrier.items.add(file);
        input.files = carrier.files;
        enctypeFor();
        if (input.files.length) show(file);
      } catch {
      }
    })();

    enctypeFor();
  })();

  // The worker is what makes this installable and what answers when the
  // network is gone. Registered from here rather than inline in the page so
  // there is one script to read, and resolved relative to this file so it does
  // not need to be told where the screen is mounted.
  if ("serviceWorker" in navigator && document.currentScript) {
    const sw = new URL("../sw.js", document.currentScript.src);
    // No scope option. The worker is served from /sw.js, so its default scope
    // is the directory it came from, which is the root — every screen. Naming
    // a scope here is how the previous version ended up claiming whichever
    // page happened to register it.
    // Anything the worker is holding goes in when the network comes back.
    // Both signals, because "online" fires on a network that is present but
    // not yet working, and a page load is the other moment worth trying.
    const flush = () => navigator.serviceWorker.ready
      .then(reg => reg.active?.postMessage("flush"))
      .catch(() => {});
    addEventListener("online", flush);
    flush();

    navigator.serviceWorker.register(sw)
      .then(subscribe)
      .catch(() => {
        // An install that fails costs the offline page and nothing else. The
        // screen is a network thing; this was always the extra.
      });
  }

  // Where to reach you when you are not looking at the screen.
  //
  // Only asked for after you press the button, and the button only exists when
  // there is a key to subscribe with. A permission prompt on page load is the
  // rudest thing a web page can do, and this one is asking to interrupt
  // someone specifically because they are bad at being interrupted — so it has
  // to be a thing you went and turned on.
  //
  // Re-subscribing on every load is deliberate and cheap: a push subscription
  // expires without telling anyone, and the endpoint is upserted, so the only
  // cost of doing it again is one request that changes nothing.
  async function subscribe(registration) {
    const key = document.body.dataset.pushKey;
    if (!key || !("PushManager" in window)) return;
    if (Notification.permission !== "granted") return;

    try {
      const sub = await registration.pushManager.subscribe({
        userVisibleOnly: true,
        applicationServerKey: keyBytes(key)
      });
      await fetch("/push/subscribe", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(sub)
      });
    } catch {
      // A subscription that cannot be made costs the fast channel and nothing
      // else: every message this would carry still reaches the room.
    }
  }

  // The base64url the server hands over, as the bytes the browser wants.
  function keyBytes(key) {
    const padded = (key + "=".repeat((4 - key.length % 4) % 4))
      .replace(/-/g, "+").replace(/_/g, "/");
    const raw = atob(padded);
    return Uint8Array.from(raw, c => c.charCodeAt(0));
  }

  // Notifications, in the settings panel.
  //
  // It was a floating button reading "tell me when to leave" that hid itself
  // for ever the moment it was answered either way — so once you had said yes
  // there was nothing that said it was on, and once you had said no there was
  // nothing at all. A setting you cannot read is not a setting.
  //
  // The server says whether it would send to anything; only the browser knows
  // whether the permission was refused, and a refusal cannot be re-asked by a
  // site. So the panel is drawn from the record and corrected here.
  function theSetting() {
    const bit = document.getElementById("pushbit");
    if (!bit) return;
    const says = document.getElementById("pushsays");
    const on = document.getElementById("pushon");
    const off = document.getElementById("pushoff");
    const key = document.body.dataset.pushKey;

    const can = key && "Notification" in window && "PushManager" in window;
    if (!can) {
      says.textContent = "This browser cannot take notifications.";
      on.hidden = off.hidden = true;
      return;
    }
    if (Notification.permission === "denied") {
      // Once a browser has been told no it will not ask again and this site
      // cannot re-ask, so a button here would be a control that cannot work.
      // What is offered instead is the one thing that can change it: where the
      // switch is. Issue #147 — before this, a no was the end of it in silence.
      says.textContent = "Blocked by this browser. Turn notifications on for " +
        "this site in its own settings and I can tell you when to leave.";
      on.hidden = off.hidden = true;
      return;
    }
    // Told, and still permitted: the record and the browser agree.
    if (bit.dataset.state === "on" && Notification.permission === "granted") {
      on.hidden = true;
      off.hidden = false;
      return;
    }
    on.hidden = false;
    off.hidden = true;

    on.addEventListener("click", async () => {
      if (Notification.permission === "default" &&
          await Notification.requestPermission() !== "granted") {
        theSetting();
        return;
      }
      await subscribe(await navigator.serviceWorker.ready);
      says.textContent = "On. Squirrel can tell you when to leave for something.";
      on.hidden = true;
      off.hidden = false;
    });

    off.addEventListener("click", async () => {
      // Both halves, or neither works. The browser's own subscription is the
      // only thing that can stop a notification arriving; the row is the only
      // thing that stops one being sent. Dropping one and not the other leaves
      // either a notification from nowhere or a row sent to for ever.
      try {
        const reg = await navigator.serviceWorker.ready;
        await (await reg.pushManager.getSubscription())?.unsubscribe();
      } catch {
        // The browser would not let go. The server half still stops the
        // sending, which is the half that decides whether anything arrives.
      }
      await fetch("/push/forget", { method: "POST", credentials: "same-origin" });
      says.textContent = "Off. Squirrel will not interrupt you.";
      off.hidden = true;
      on.hidden = false;
    });
  }
  theSetting();
})();
