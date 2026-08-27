// Progressive enhancement, and nothing here is load-bearing. Every action on
// this page is a form submission or a link that works with this file absent;
// what this adds is a slot that grows with what you type, a photograph held
// somewhere durable the moment it is chosen, one key per action on the chores,
// and the worker that makes the whole thing installable.
(() => {
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

  // ---- the photograph in the slot -----------------------------------------
  //
  // Two things, and they are the same thing said twice: a photograph you have
  // chosen has to be visible, and it has to still be there when you come back.
  //
  // The input is a pixel wide and invisible, because a file input cannot be
  // styled. So choosing one changed nothing on the screen — the camera looked
  // identical before and after, and the only way to know whether the next
  // press would keep a photograph was to press it and go and look.
  //
  // Worse, and this is the part that made it "saving doesn't work": choosing
  // one on a phone hands the screen to another app, and an installed app that
  // is handed away can be reclaimed while it waits. It comes back reloaded.
  // The input is empty, the page looks exactly as it did before — because it
  // never looked any different — and the press that follows keeps the words
  // with no photograph attached.
  //
  // So the photograph goes somewhere durable the moment it is chosen, which is
  // the same answer the spool is for the same problem one layer down: hold it
  // before anything says it was kept. IndexedDB rather than a variable,
  // because a variable is exactly what the reload takes away.
  //
  // Its own database rather than the worker's: that one is at version 1 with
  // one store in it, and adding a second from here would mean a version bump
  // racing a worker that has its own idea of the schema.
  //
  // Enhancement only. With this file absent the input is a file input in a
  // multipart form, which posts a photograph perfectly well and shows you
  // nothing — the floor, and it is the floor this was built on.
  (() => {
    // The capture slot on home, camera or no camera. Two things live here and
    // only one of them needs a camera: holding a chosen photograph, and
    // keeping a capture without the page going anywhere.
    const form = document.querySelector('form.slot[action="/capture"]');
    if (!form) return;

    // DataTransfer is how a file gets back onto an input. Without it a
    // restored photograph could be shown and not sent, which is a screen that
    // lies about what it has — worse than one that shows nothing. No camera
    // and no DataTransfer are both fine; the slot still keeps things.
    const input = typeof DataTransfer === "undefined"
      ? null : form.querySelector('input[name="photo"]');

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

    // What you are about to keep, under the words rather than beside them: it
    // is the note and the box above it is the caption, which is the order the
    // card puts them in too.
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
    if (input) form.append(shown);

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

    // Multipart only when there is actually a photograph in it.
    //
    // The worker forwards multipart captures straight to the network, because
    // the only thing it can hold offline is the words and a photograph would
    // be silently dropped out of one. A slot that always claimed multipart
    // would therefore give up the offline hold on every words-only capture,
    // which is the case the hold exists for.
    function enctypeFor() {
      form.enctype = input?.files?.length
        ? "multipart/form-data"
        : "application/x-www-form-urlencoded";
    }

    if (input) input.addEventListener("change", () => {
      const file = input.files?.[0];
      enctypeFor();
      if (!file) { hide(); forget().catch(() => {}); return; }
      show(file);
      // Deliberately not awaited before the picture appears: the screen must
      // say it has the photograph the instant it does, and a write that fails
      // costs the durability rather than the photograph in front of you.
      //
      // Which leaves a window — chosen, drawn, not yet held — where losing the
      // page loses the picture. It is the few milliseconds an IndexedDB write
      // takes, and closing it would mean holding the preview back until the
      // disk agreed, which trades a visible delay on every photograph for a
      // race nobody can hit on purpose. Stated rather than hidden: CI lost
      // this race once, which is how it came to be written down.
      stash(file).catch(() => {});
    });

    off.addEventListener("click", () => {
      if (input) input.value = "";
      enctypeFor();
      hide();
      forget().catch(() => {});
    });

    // On the way in. A capture that landed says so in the URL, and that is the
    // one case where the photograph in the stash is one already kept — so it
    // is dropped rather than offered back, which would keep it twice.
    if (input) (async () => {
      try {
        if (new URLSearchParams(location.search).has("kept")) {
          await forget();
          return;
        }
        const file = await stashed();
        if (!file) return;
        // Put it back on the input before showing it, so the screen never
        // claims a photograph the next press would not send.
        const carrier = new DataTransfer();
        carrier.items.add(file);
        input.files = carrier.files;
        enctypeFor();
        if (input.files.length) show(file);
      } catch {
        // No IndexedDB, a private window, a browser that will not take files
        // back: the slot is a slot and the input is an input, which is where
        // this started.
      }
    })();

    // A slot that starts empty starts urlencoded, whatever the markup said.
    // Without this the first capture of a session — the words-only one, typed
    // before any photograph has been chosen — would still go out as multipart
    // and lose its offline hold.
    enctypeFor();

    // ---------------------------------------------------------------- //
    // Keeping something, without the page going anywhere.
    //
    // Posting the form navigates: the browser leaves, the server answers 303,
    // the page comes back and you are at the top of it reading a small word
    // that is nowhere near the box you typed in. Two things go wrong there and
    // neither is the capture — you lose your place, and the answer to "did
    // that work?" is somewhere other than the thing you used.
    //
    // So the script posts it and stays put. The box empties, the button says
    // it landed, and nothing on the screen moves. What the server thinks is
    // still the only opinion that counts: the outcome is read out of the URL
    // it redirects to, in exactly the vocabulary the scriptless path uses, so
    // there is one set of answers rather than two that can disagree.
    //
    // Enhancement only. With this file gone the form posts, the page reloads,
    // and the same words appear in the same element — slower, and correct.
    // ---------------------------------------------------------------- //
    const said = form.querySelector("#slotsaid");
    const post = form.querySelector(".post");
    const box = form.querySelector("textarea");
    if (!said || !post) return;

    // The thread is the same slot in a different room, and this file keeps
    // owning it: the camera, the stash, the survival of the app being
    // reclaimed and of the worker taking the words offline all live here, and
    // handing capture to another file would have to move all of it. What
    // changes on the thread is only what the answer looks like — a turn
    // appended to the conversation instead of a word inside the box.
    const thread = document.getElementById("thread");

    // Word for word what the server renders, because the comment below has
    // always claimed they were the same and they were not: this path dropped
    // "try again in a moment" and "keep them without it, or try another
    // picture" — the sentence that says what to do — from both failures, on
    // the path nearly every session takes. A test pins them together now.
    const words = {
      kept: "kept",
      held: "No network — I have it. It goes in when you are back.",
      nokeep: "Not kept — Squirrel cannot reach its memory. Your words are still here; try again in a moment.",
      nophoto: "That photograph was not kept — too big, or a kind Squirrel does not take. Your words are still here; keep them without it, or try another picture.",
    };
    let telling = 0;

    function tell(which) {
      said.textContent = words[which] || words.nokeep;
      said.className = "slotsaid" + (which === "kept" ? "" : which === "held" ? " held" : " bad");
      said.hidden = false;
      clearTimeout(telling);
      // The good news goes; the bad news stays until something else happens.
      // A failure you have to be quick to read is a failure you will meet
      // again without knowing why.
      if (which === "kept") telling = setTimeout(() => { said.hidden = true; }, 4000);
    }

    form.addEventListener("submit", async e => {
      // A browser with no fetch, or a form somehow without an action: let it
      // navigate, which is the floor and works.
      if (typeof fetch !== "function") return;
      e.preventDefault();
      if (post.disabled) return;

      const carrying = !!input.files?.length;
      // The same bytes the form itself would have sent. FormData when there is
      // a file, because that is what multipart is for; URLSearchParams when
      // there is not, so the worker can still hold it offline.
      const body = carrying ? new FormData(form)
        : new URLSearchParams([...new FormData(form)].filter(([, v]) => typeof v === "string"));
      const init = { method: "POST", body, credentials: "same-origin", headers: {} };
      if (!carrying) init.headers["Content-Type"] = "application/x-www-form-urlencoded";
      // Ask for the turns rather than a redirect. The server renders them from
      // the same templates the page uses, so there is one description of a
      // card rather than two that can disagree.
      if (thread) init.headers["X-Thread"] = "fragment";

      post.disabled = true;
      try {
        const res = await fetch(form.action, init);

        if (thread) {
          // What came back is the conversation's next two turns, as HTML from
          // this server's own templates — see thread.js for why parsing it is
          // safe. Empty means the server decided there was nothing to keep.
          const html = res.ok ? await res.text() : "";
          if (html.trim()) window.__threadAppend?.(html);
          // Only when the server said it kept them. A failure comes back as
          // turns too, and emptying the box on one of those is a capture box
          // that eats thoughts — which is the thing this whole path exists
          // not to do.
          if (res.headers.get("X-Kept") === "1") {
            box.value = "";
            box.style.height = "auto";
            if (input.files?.length) { input.value = ""; enctypeFor(); hide(); forget().catch(() => {}); }
          }
          post.disabled = false;
          return;
        }

        const out = new URL(res.url, location.href).searchParams;
        const which = ["kept", "held", "nokeep", "nophoto"].find(k => out.has(k));
        // Nothing at all in the URL means the server decided there was nothing
        // to keep — an empty box, pressed. Say nothing back; it did nothing.
        if (!which) { post.disabled = false; return; }

        tell(which);
        if (which === "kept" || which === "held") {
          box.value = "";
          box.style.height = "auto";
          if (input.files?.length) { input.value = ""; enctypeFor(); hide(); forget().catch(() => {}); }
        }
      } catch {
        // The network went while it was in the air. The words and the
        // photograph are both still on the screen, which is the whole reason
        // this box never clears until the server has said so.
        if (thread) window.__threadSay?.(words.nokeep);
        else tell("nokeep");
      } finally {
        post.disabled = false;
      }
    });
  })();

  // ---- the chores screen -------------------------------------------------
  //
  // A letter has to know which chore it means, and rather than invent a
  // selection model this uses the platform's own: the chore you are focused in
  // is the chore a letter acts on. That keeps DESIGN.md's rule intact —
  // letters are actions, movement is the arrow keys.
  //
  // With this file absent every button is still a button and Tab still
  // reaches it. Nothing below is load-bearing.
  // Asked for each time rather than collected once at load.
  //
  // The chores arrive in the conversation now, after the page has painted, so a
  // list taken at load is empty for exactly the cards these keys are for. The
  // query costs nothing at the moment a letter is pressed.
  const allChores = () => [...document.querySelectorAll("article.chore")];

  function focusedChore() {
    return document.activeElement?.closest?.("article.chore") || null;
  }

  function moveChore(step) {
    const chores = allChores();
    if (!chores.length) return;
    const here = focusedChore();
    const at = here ? chores.indexOf(here) : -1;
    const next = chores[Math.min(Math.max(at + step, 0), chores.length - 1)];
    next?.querySelector("button, summary")?.focus();
  }

  // `o` was a disclosure inside the card; the interval question is a turn of
  // its own now, and HOW OFTEN is the button that asks for it.
  const CHORE_KEYS = { d: ".abtn.did", s: ".abtn.stop", o: ".abtn.go" };

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

  // Anything you can type into owns the keys it is given.
  //
  // Every letter on this screen is an action — d is done, s is stop, c asks
  // for an interval — and until this existed those actions fired while you
  // were typing. Naming a chore "shopping" pressed stop, then opened the
  // interval question, and each press moved the focus, which on a phone shuts
  // the keyboard. The reword box and the slot had the same problem.
  //
  function typing(target) {
    if (!target) return false;
    return target.isContentEditable === true ||
      ["INPUT", "TEXTAREA", "SELECT"].includes(target.tagName);
  }

  addEventListener("keydown", e => {
    // A modal is modal for the keyboard too. Nothing in this product is a
    // <dialog> since Buddy's sheet went on 25 August 2026, and the check stays
    // because the rule is about any modal rather than about that one — with one open
    // and focus on any control that is not a text box, pressing `d` stamped
    // the card underneath, invisibly, because a modal was over it. On the
    // chores that same press could fire DID IT or STOP ASKING on whichever
    // chore held the roving focus.
    //
    // `typing()` cannot cover this: it exempts text boxes, and the sheet's
    // cross, its chips and its send button are none of those.
    //
    // The whole handler stands down rather than only the letters, and it does
    // so wherever the focus is. A key pressed *inside* the sheet is the
    // reported case — the sheet has no letter actions of its own, so there is
    // nothing here for it to want, and "/" jumping you into a search field
    // behind a modal is the same bug wearing a different key.
    if (document.querySelector("dialog[open]")) return;
    if (typing(e.target)) return;
    // Capture, on the key beside the one that finds. Looking something up had
    // a keyboard path and keeping a thought did not, so on the deliberate
    // desktop the first thing home asked of a keyboard was to reach for the
    // mouse — for the one act this product calls sacred.
    //
    // `t` because it is the verb the button already uses: Tell it. It reaches
    // the slot wherever the slot is, which is home and the ladder's own
    // capture box, and does nothing on a screen that has neither.
    if (e.key === "t" || e.key === "T") {
      const box = document.querySelector(".slot textarea");
      if (box) { e.preventDefault(); box.focus(); return; }
    }
    // A focused control owns space and enter; that is the platform's contract.
    if ((e.key === " " || e.key === "Enter") && e.target.closest("button, summary, a")) return;

    // Asked for rather than captured at load: the chores arrive in the
    // conversation after the page has painted, and a list taken at load is
    // empty for exactly the cards these keys are for.
    if (allChores().length) {
      // A question in progress owns the arrows. Without this carve-out an
      // arrow aimed at the picker's chips threw the focus onto a different
      // chore's buttons — and the next letter acted on that one.
      const asking = document.querySelector(".pick");
      if (asking && asking.contains(document.activeElement)) {
        if (e.key === "ArrowDown" || e.key === "ArrowUp") { e.preventDefault(); return; }
      }
      if (e.key === "ArrowDown") { e.preventDefault(); moveChore(1); return; }
      if (e.key === "ArrowUp") { e.preventDefault(); moveChore(-1); return; }

      const key = e.key.toLowerCase();
      if (key in CHORE_KEYS) {
        e.preventDefault();
        choreKey(key);
      }
      return;
    }

  });


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
  const refused = document.getElementById("pushRefused");
  if (askPush) {
    // Once a browser has been told no it will not ask again, and this site
    // cannot re-ask. A button would be a control that cannot work, so what is
    // shown instead is the only thing that can change it: where the switch is.
    // Issue #147 — before this, a no was the end of it with nothing said.
    if (refused && document.body.dataset.pushKey && "Notification" in window &&
        Notification.permission === "denied") {
      refused.hidden = false;
    }
    if (!document.body.dataset.pushKey || !("Notification" in window) ||
        Notification.permission !== "default") {
      askPush.hidden = true;
    } else {
      // Taking `hidden` off is the whole of this control working, and it was
      // missing. The template ships the button hidden — which is right, because
      // a control that flashes on and then decides it should not be there is
      // worse than one that arrives a moment late — and this branch attached a
      // listener to something nobody could press. Permission was therefore
      // never requested on any device, `push_subscriptions` held zero rows, and
      // every leave-by warning fanned out over an empty list in silence.
      //
      // The test that covered this asserted the button was in the markup. It
      // was. See askpush_test.go.
      askPush.hidden = false;
      askPush.addEventListener("click", async () => {
        askPush.hidden = true;
        if (await Notification.requestPermission() !== "granted") {
          // Said now rather than on the next visit: the moment you refused is
          // the moment the sentence is about something you just did.
          if (refused) refused.hidden = false;
          return;
        }
        const registration = await navigator.serviceWorker.ready;
        await subscribe(registration);
      });
    }
  }

})();
