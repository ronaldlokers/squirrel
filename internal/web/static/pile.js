// Progressive enhancement, and nothing here is load-bearing. Every action on
// this page is a form submission or a link that works with this file absent;
// what this adds is the stamp, the moment the card holds still so the undo has
// somewhere to be, one key per action, and search that answers as you type.
(() => {
  const find = document.querySelector(".find input");
  const stage = document.getElementById("stage");
  const say = document.getElementById("say");

  // Anything this file changes without a navigation has to be said out loud
  // too, or the screen is silent for exactly the person who cannot see it
  // change. The region lives outside the stage so a search swap cannot take it
  // away mid-announcement.
  function announce(what) {
    if (say) say.textContent = what;
  }

  // Honoured here as well as in the stylesheet: the CSS can shorten an
  // animation, but the pause that lets the undo be read is this file's, and
  // someone who has asked for less motion should not be made to sit through
  // a card sliding away before the write happens.
  const calm = matchMedia("(prefers-reduced-motion: reduce)");
  const hold = () => (calm.matches ? 400 : 1150);
  const leave = () => (calm.matches ? 0 : 440);

  const STATES = {
    done:  { word: "DONE",    said: "marked done" },
    keep:  { word: "KEPT",    said: "kept as reference" },
    drop:  { word: "DROPPED", said: "dropped" },
    chore: { word: "CHORE",   said: "now a chore" }
  };

  // Everything below hangs off whatever is currently in #stage. Live search
  // replaces that wholesale — a search can turn into a deck and back — so the
  // wiring is a function that runs again on the new markup rather than a set
  // of listeners bound once at load.
  let deck = null;

  function wire() {
    const card = document.getElementById("card");
    deck = card ? bindCard(card) : null;
    scatter();
  }

  // Habituation is the enemy: the stack never sits the same way twice.
  function scatter() {
    document.querySelectorAll(".behind").forEach((el, n) => {
      const rot = (n % 2 ? -1 : 1) * (0.7 + Math.random() * 1.3);
      const step = 1 + n * 0.8 + Math.random() * 0.5;
      el.style.transform =
        `translate(calc(var(--o) * ${step.toFixed(2)}), calc(var(--o) * ${(step * 1.1).toFixed(2)})) rotate(${rot.toFixed(2)}deg)`;
    });
  }

  function bindCard(card) {
    const form = document.getElementById("actions");
    const stamp = document.getElementById("stamp");
    const stampText = document.getElementById("stampText");
    const undoRow = document.getElementById("undoRow");
    const said = document.getElementById("said");
    const undo = document.getElementById("undo");
    const every = form.querySelector("details.everyFallback");
    const chips = [...form.querySelectorAll("button[name=every]")];
    const neverMind = form.querySelector("[data-close=chore]");
    let going = false;

    // Act, then hold, then submit. The delay is not decoration: it is the
    // moment the spec asks for, and the undo has to be reachable while the
    // card that it undoes is still on the screen.
    function go(button) {
      if (going) return;
      going = true;
      const kind = button.dataset.act || "done";
      const s = STATES[kind] || STATES.done;
      const token = kind === "keep" ? "kept" : kind === "drop" ? "dropped" : kind;
      stamp.style.setProperty("--sc", `var(--${token})`);
      stamp.style.setProperty("--sct", `var(--${token}-ink)`);
      stampText.textContent = s.word;
      said.textContent = s.said;
      card.classList.add("stamped");
      form.hidden = true;
      undoRow.hidden = false;
      undo.focus({ preventScroll: true });
      announce(s.said + ". Put it back is focused.");

      setTimeout(() => {
        card.classList.add("leaving");
        setTimeout(() => {
          // Hand the real submission back to the browser. The server does the
          // write and answers 303, so a reload never repeats it.
          button.form.requestSubmit(button);
        }, leave());
      }, hold());
    }

    // Choosing an interval takes the place of the actions rather than sitting
    // under them, the way the comp draws it. The disclosure already does the
    // showing; the class does the hiding, and neither exists without this file.
    function choosing(open) {
      if (open !== every.open) announce(open ? "how often should it come back?" : "never mind");
      every.open = open;
      form.classList.toggle("choosing", open);
      neverMind.hidden = !open;
      if (open) chips[0]?.focus({ preventScroll: true });
      else every.querySelector("summary")?.focus({ preventScroll: true });
    }

    every.addEventListener("toggle", () => choosing(every.open));
    neverMind?.addEventListener("click", () => choosing(false));

    form.addEventListener("click", e => {
      const b = e.target.closest("button[name=act], button[name=every]");
      if (!b || going) return;
      e.preventDefault();
      go(b);
    });

    // PUT IT BACK is the transition back to open, posted like any other.
    undo.addEventListener("click", () => {
      const back = document.createElement("form");
      back.method = "post";
      back.action = form.action;
      // Built as nodes rather than as markup: the id came from the page, but a
      // string spliced into innerHTML is a habit that outlives the one safe
      // value it was written for.
      for (const [name, value] of [["id", card.dataset.id], ["act", "open"]]) {
        const field = document.createElement("input");
        field.type = "hidden";
        field.name = name;
        field.value = value;
        back.append(field);
      }
      document.body.append(back);
      back.submit();
    });

    return {
      get going() { return going; },
      card, form, every, chips, choosing,
      act(name) {
        const b = form.querySelector(`[data-act="${name}"]`);
        if (b) go(b);
      },
      chip(n) { if (chips[n]) go(chips[n]); }
    };
  }

  // ---- live search --------------------------------------------------------
  //
  // The same URL the form already submits to, fetched and swapped in. One
  // renderer, one code path: what lands here is exactly the page a full
  // navigation would have produced, which is why nothing about the scriptless
  // path changes.

  let pending = null;
  let timer = 0;
  let shown = new URL(location.href).searchParams.get("q") || "";

  async function swap(query) {
    const url = new URL(location.href);
    if (query) url.searchParams.set("q", query);
    else url.searchParams.delete("q");
    // A search is a fresh look at everything, so it drops the skip position
    // rather than searching from halfway down the pile.
    url.searchParams.delete("after");
    url.searchParams.delete("undo");
    url.searchParams.delete("was");
    url.searchParams.delete("state");

    pending?.abort();
    pending = new AbortController();
    let html;
    try {
      const res = await fetch(url, { signal: pending.signal, headers: { "X-Requested-With": "fetch" } });
      if (!res.ok) return;                 // leave what is on the screen alone
      html = await res.text();
    } catch (e) {
      return;                              // aborted, or the network went away
    }

    const fresh = new DOMParser().parseFromString(html, "text/html").getElementById("stage");
    if (!fresh) return;
    stage.innerHTML = fresh.innerHTML;
    stage.className = fresh.className;
    shown = query;
    // replaceState, not pushState: one entry for the search rather than one
    // per keystroke, so back goes where you came from instead of through every
    // letter you typed.
    history.replaceState(null, "", url);
    wire();
    // What changed, in words, since nothing navigated and the eye has the
    // whole page to notice it with. Never how many — the rule holds here as
    // everywhere.
    announce(query ? `showing everything that says ${query}` : "the pile");
  }

  if (find) {
    find.form.addEventListener("submit", e => {
      // Enter is already what the field does; intercepting it keeps the page
      // from reloading under a search that is already on screen.
      e.preventDefault();
      clearTimeout(timer);
      swap(find.value.trim());
    });
    find.addEventListener("input", () => {
      clearTimeout(timer);
      const value = find.value.trim();
      if (value === shown) return;
      timer = setTimeout(() => swap(value), 180);
    });
  }

  // ---- keys ---------------------------------------------------------------
  //
  // Letters are actions, always. Movement is space and the arrows, because in
  // a one-card topology j/k has nothing to move between and k is keep.

  // T decides: not a disposal, but the same one-key gesture.
  const KEYS = { d: "done", k: "keep", x: "drop", t: "task" };

  // The key presses the card's own LATER link rather than working out where
  // to go: one answer to "what does skipping mean", and it is the one a phone
  // and a scriptless page already use.
  function skip() {
    announce("skipped");
    const later = deck?.card.querySelector("a.later");
    if (!later) return;
    const url = new URL(later.href);
    for (const stale of ["undo", "was", "state"]) url.searchParams.delete(stale);
    location.assign(url);
  }


  // ---- the slot -----------------------------------------------------------
  //
  // The field grows with what is in it. Without this a thought longer than one
  // line is typed into a one-line box, and — worse — words handed back after a
  // failed write arrive clipped, which reads as "some of it is gone" at exactly
  // the moment the page is promising the opposite.
  //
  // Enhancement only: with this file absent the textarea scrolls, which is
  // ugly and loses nothing.
  const slot = document.querySelector(".slot textarea");
  if (slot) {
    const grow = () => {
      slot.style.height = "auto";
      slot.style.height = slot.scrollHeight + "px";
    };
    slot.addEventListener("input", grow);
    grow();

    // Enter keeps it, the way Enter sends a message in the room this product
    // lives in. Shift+Enter is the newline, for a thought with two parts.
    slot.addEventListener("keydown", e => {
      if (e.key !== "Enter" || e.shiftKey) return;
      e.preventDefault();
      if (slot.value.trim()) slot.form.requestSubmit();
    });
  }

  // ---- the chores screen -------------------------------------------------
  //
  // The deck has one card, so a key there needs no idea of which thing it
  // means. A list does, and rather than invent a selection model this uses the
  // platform's own: the chore you are focused in is the chore a letter acts
  // on. That also keeps DESIGN.md's rule intact — letters are actions, and
  // movement is the arrow keys — applied to a list rather than to a deck.
  //
  // With this file absent every button is still a button and Tab still
  // reaches it. Nothing below is load-bearing.
  const chores = [...document.querySelectorAll("article.chore")];

  function focusedChore() {
    return document.activeElement?.closest?.("article.chore") || null;
  }

  function moveChore(step) {
    if (!chores.length) return;
    const here = focusedChore();
    const at = here ? chores.indexOf(here) : -1;
    const next = chores[Math.min(Math.max(at + step, 0), chores.length - 1)];
    next?.querySelector("button, summary")?.focus();
  }

  const CHORE_KEYS = { d: ".abtn.did", s: ".abtn.stop", o: "details.often > summary" };

  function choreKey(key) {
    const card = focusedChore();
    if (!card) { moveChore(1); return true; }
    const target = card.querySelector(CHORE_KEYS[key]);
    if (!target) return false;
    // Pressing the control rather than submitting behind its back: one answer
    // to what each key means, and it is whatever the button already did.
    target.click();
    // Opening the question hides the two actions beside it, and one of them
    // was holding the focus — which drops focus to the body and leaves the
    // next key with no chore to act on. The summary survives the open (it
    // becomes "never mind"), so it is where focus belongs.
    if (document.activeElement === document.body) target.focus();
    return true;
  }

  addEventListener("keydown", e => {
    if (e.target === find) {
      if (e.key === "Escape") { find.value = ""; clearTimeout(timer); swap(""); }
      return;
    }
    if (e.key === "/") { e.preventDefault(); find?.focus(); return; }
    // A focused control owns space and enter; that is the platform's contract.
    if ((e.key === " " || e.key === "Enter") && e.target.closest("button, summary, a")) return;

    // The chores screen. Its own keys, because it is a list and the deck is
    // not — and its own branch, because there is no card here to act on.
    if (chores.length) {
      if (e.key === "ArrowDown") { e.preventDefault(); moveChore(1); return; }
      if (e.key === "ArrowUp") { e.preventDefault(); moveChore(-1); return; }
      // Withdrawing the interval question is Escape here as it is on the deck.
      if (e.key === "Escape") {
        // The focused chore's question, or whichever is open if focus has
        // wandered off — Escape means "withdraw the question" either way.
        const open = focusedChore()?.querySelector("details.often[open]")
          || document.querySelector("details.often[open]");
        if (open) {
          e.preventDefault();
          open.open = false;
          open.querySelector("summary")?.focus();
        }
        return;
      }
      const key = e.key.toLowerCase();
      if (key in CHORE_KEYS) {
        e.preventDefault();
        choreKey(key);
      }
      return;
    }

    if (!deck || deck.going) return;

    // The chore interval is a question, not an action: C asks it, 1-4 answer
    // it, and ESC withdraws it. Nothing here fires a write on its own.
    if (deck.every?.open) {
      const n = "1234".indexOf(e.key);
      if (n >= 0) { e.preventDefault(); deck.chip(n); return; }
      if (e.key === "Escape") { e.preventDefault(); deck.choosing(false); }
      return;
    }
    if (e.key.toLowerCase() === "c") { e.preventDefault(); deck.choosing(true); return; }

    if (e.key === " " || e.key === "ArrowRight" || e.key === "ArrowDown") {
      e.preventDefault();
      skip();
      return;
    }
    if (e.key === "ArrowLeft" || e.key === "ArrowUp") { e.preventDefault(); history.back(); return; }

    const act = KEYS[e.key.toLowerCase()];
    if (act) { e.preventDefault(); deck.act(act); }
  });

  wire();

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

  // The button, which only appears when there is a key and permission has not
  // been answered. Once it has been answered — either way — it goes: a control
  // that cannot change anything is furniture, and one that asks again after a
  // no is a nag.
  const askPush = document.getElementById("askPush");
  if (askPush) {
    if (!document.body.dataset.pushKey || !("Notification" in window) ||
        Notification.permission !== "default") {
      askPush.hidden = true;
    } else {
      askPush.addEventListener("click", async () => {
        askPush.hidden = true;
        if (await Notification.requestPermission() !== "granted") return;
        const registration = await navigator.serviceWorker.ready;
        await subscribe(registration);
      });
    }
  }
})();
