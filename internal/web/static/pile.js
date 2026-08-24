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

  // The three ways of not being able to act on something. They share one
  // stamp colour: which of the three it is says why, not what happened.
  const HELD = { waiting: 1, blocked: 1, someday: 1 };

  const STATES = {
    done:  { word: "DONE",    said: "marked done" },
    keep:  { word: "KEPT",    said: "kept as reference" },
    drop:  { word: "DROPPED", said: "dropped" },
    chore: { word: "CHORE",   said: "now a chore" },
    // Deciding is not disposing, and this was the one answer with no entry
    // here — so it fell through to `STATES.done` and a note promoted to a task
    // stamped itself DONE.
    task:  { word: "A TASK",  said: "now a task" },
    // Set aside. The same three the server knows, so the stamp on the card and
    // the line on the next page say the same thing for the same press — which
    // is the rule saidWords exists to keep.
    waiting: { word: "WAITING", said: "waiting on someone" },
    blocked: { word: "BLOCKED", said: "blocked on a thing" },
    someday: { word: "SOMEDAY", said: "someday" }
  };

  // Everything below hangs off whatever is currently in #stage. Live search
  // replaces that wholesale — a search can turn into a deck and back — so the
  // wiring is a function that runs again on the new markup rather than a set
  // of listeners bound once at load.
  let deck = null;

  // A tapped action is written ~1150ms after the press, so the undo has a card
  // to sit on. Three things can take the page away inside that window — the
  // way out of the deck, a live search replacing the stage, and the phone
  // being locked or the app backgrounded — and in all three the write was
  // simply lost, on a screen that had already shown you the stamp.
  //
  // `owed` is that write, held where anything that takes the page can settle
  // it first. It clears itself, so settling twice is not a second write.
  let owed = null;

  // The ordinary way: hand the submission back to the browser, which navigates
  // and gets the 303. Only works while the form is still in the document.
  function settle() {
    const o = owed;
    owed = null;
    if (!o) return;
    o.stop();
    o.form.requestSubmit(o.button);
  }

  // The last-moment way, for a page that is going whether or not the write is
  // done: the same bytes, sent with `keepalive` so the browser finishes it
  // after the document is gone. No navigation to wait for and none wanted —
  // the answer is a 303 nobody will read.
  function flush() {
    const o = owed;
    owed = null;
    if (!o) return;
    o.stop();
    fetch(o.form.action, {
      method: "POST",
      headers: { "Content-Type": "application/x-www-form-urlencoded" },
      body: o.body,
      keepalive: true,
      credentials: "same-origin",
    }).catch(() => {});
  }

  // Locked, switched away from, or closed. `pagehide` is the one iOS actually
  // fires when an installed app is backgrounded; `visibilitychange` covers the
  // tab going to the background without unloading. Both settle the same debt,
  // and settling twice is not a second write.
  addEventListener("pagehide", flush);
  addEventListener("visibilitychange", () => { if (document.visibilityState === "hidden") flush(); });

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
    // The hold's own timer, and the write it still owes. Both live out here
    // because two other things can take the page away before the hold is up.
    let waiting = 0;

    // Act, then hold, then submit. The delay is not decoration: it is the
    // moment the spec asks for, and the undo has to be reachable while the
    // card that it undoes is still on the screen.
    function go(button) {
      if (going) return;
      going = true;
      // `aside` for the three chips, `data-act` for everything else. The
      // stamp's colour follows the same name, and the three take the state
      // colours their own screen already uses.
      const kind = button.dataset.act || button.value || "done";
      const s = STATES[kind] || STATES.done;
      const token = kind === "keep" ? "kept" : kind === "drop" ? "dropped"
        : kind === "task" ? "violet"
        : STATES[kind] && kind in HELD ? "held" : kind;
      stamp.style.setProperty("--sc", `var(--${token})`);
      stamp.style.setProperty("--sct", `var(--${token}-ink)`);
      stampText.textContent = s.word;
      said.textContent = s.said;
      card.classList.add("stamped");
      form.hidden = true;
      // The repairs go with the answers.
      //
      // They are siblings of the form rather than children of it — correcting
      // the words is not an answer, which is exactly why it sits outside the
      // row — so hiding the form left them on a card that had just been
      // dropped. For the length of the hold you were being offered "fix the
      // words" and "i can't act on this" about a note that was already gone,
      // and "this is more than one thing" about a note that was no longer any
      // things at all.
      //
      // What a stamped card shows is the stamp and the way back. Nothing else
      // on it means anything now.
      card.querySelectorAll(".ways, .cantact, form:has(.askSplit)")
        .forEach((el) => { el.hidden = true; });
      undoRow.hidden = false;
      undo.focus({ preventScroll: true });
      announce(s.said + ". Put it back is focused.");

      // The way out stops being a way out. LATER is a link, so it navigates
      // the moment it is pressed — and it sat live and visible through the
      // whole hold, next to a card that had already been stamped. Pressing it
      // took the page away before the write left, and the decision you had
      // watched happen was simply gone.
      card.querySelector("#later")?.setAttribute("hidden", "");

      // What is owed to the server, if anything takes the page first. The body
      // is built now, while the form is certainly still attached and certainly
      // still says what was pressed.
      const data = new FormData(button.form);
      if (button.name) data.append(button.name, button.value);
      owed = {
        form: button.form,
        button,
        body: new URLSearchParams([...data].filter(([, v]) => typeof v === "string")),
        stop() { clearTimeout(waiting); },
      };

      waiting = setTimeout(() => {
        card.classList.add("leaving");
        waiting = setTimeout(settle, leave());
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

    // Setting one aside is a disposition like the other four and now leaves
    // like one. Its three chips live in their own form, outside `.actions`,
    // because "i can't act on this" is not an answer to what the note is — so
    // they were never wired to any of this, and a note set aside vanished with
    // no stamp, no hold and nothing offering it back.
    card.querySelector(".whys")?.addEventListener("click", e => {
      const b = e.target.closest("button[name=aside]");
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
    // Anything owed goes now, while the form that owes it is still in the
    // document. Replacing the stage detaches that form, and a detached form
    // does not submit — the browser refuses silently, so the press was kept,
    // stamped, announced, and then dropped with nothing said.
    settle();
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
  // The deck has one card, so a key there needs no idea of which thing it
  // means. A list does, and rather than invent a selection model this uses the
  // platform's own: the chore you are focused in is the chore a letter acts
  // on. That also keeps DESIGN.md's rule intact — letters are actions, and
  // movement is the arrow keys — applied to a list rather than to a deck.
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
  // The search field is not covered here because it has its own branch below:
  // Escape means "clear the search" there, and that is worth keeping.
  function typing(target) {
    if (!target || target === find) return false;
    return target.isContentEditable === true ||
      ["INPUT", "TEXTAREA", "SELECT"].includes(target.tagName);
  }

  addEventListener("keydown", e => {
    // A modal is modal for the keyboard too. Buddy's sheet is a <dialog>, and
    // every letter below acts on the screen behind it — so with the sheet open
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
    if (e.target === find) {
      if (e.key === "Escape") { find.value = ""; clearTimeout(timer); swap(""); }
      return;
    }
    // Before "/" as well, or typing a path into a box would jump you into the
    // search field.
    if (typing(e.target)) return;
    if (e.key === "/") { e.preventDefault(); find?.focus(); return; }
    // Capture, on the key beside the one that finds. Looking something up had
    // a keyboard path and keeping a thought did not, so on the deliberate
    // desktop the first thing home asked of a keyboard was to reach for the
    // mouse — for the one act this product calls sacred.
    //
    // `t` because it is the verb the button already uses: Tell it. It reaches
    // the slot wherever the slot is, which is home and the ladder's own
    // capture box, and does nothing on a screen that has neither.
    // `t` is A TASK on the deck, and per-screen meanings are how the letters
    // already work here — `d` is DONE on the deck and DID IT on the chores. So
    // this is only ever the slot's key on a screen with a slot and no deck,
    // which is checked rather than assumed: a screen that ever had both would
    // otherwise have one letter meaning two things.
    if ((e.key === "t" || e.key === "T") && !deck) {
      const box = document.querySelector(".slot textarea");
      if (box) { e.preventDefault(); box.focus(); return; }
    }
    // A focused control owns space and enter; that is the platform's contract.
    if ((e.key === " " || e.key === "Enter") && e.target.closest("button, summary, a")) return;

    // The chores. Their own keys, because they are a list and the deck is
    // not — and their own branch, because there is no card here to act on.
    //
    // Asked for rather than captured at load: the chores arrive in the
    // conversation after the page has painted, and a list taken at load is
    // empty for exactly the cards these keys are for.
    if (allChores().length) {
      // A question in progress owns the keys, exactly as it does on the deck:
      // 1-4 answer it, and moving between chores is not what an arrow means
      // while it is open.
      //
      // The deck earned those digits precisely so a picker could not be
      // interrupted by movement keys. This screen has the same picker and
      // never got the carve-out, so an arrow aimed at the four chips threw the
      // focus onto a different chore's buttons — and the next letter then
      // acted on that one.
      // The interval question is a turn of its own now rather than a
      // disclosure inside the card, so the carve-out is about the picker
      // wherever it is on the page: while one is open the arrows belong to it,
      // not to the movement between chores.
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
        if (await Notification.requestPermission() !== "granted") return;
        const registration = await navigator.serviceWorker.ready;
        await subscribe(registration);
      });
    }
  }

  // ---------------------------------------------------------------- //
  // Buddy.
  //
  // /buddy is a real page and the acorn is a real link to it; everything here
  // upgrades that into a sheet over whatever you were already looking at,
  // because the conversation is about the screen behind it and navigating away
  // from that screen is the one thing this must not do.
  //
  // A native <dialog>, so Escape closes it and focus stays inside it without
  // either being implemented here. Nothing below is load-bearing: with this
  // file absent the acorn is a link, /buddy is a page, and every form on it
  // posts and redirects.
  // ---------------------------------------------------------------- //
  (() => {
    const acorn = document.querySelector(".tobuddy");
    if (!acorn || typeof HTMLDialogElement === "undefined") return;

    let sheet = null;
    // What was typed, kept across a close. A box that clears when you close it
    // is a box that eats thoughts — the slot's rule, and it holds here for the
    // same reason. In memory only: it is a half-written sentence, not a note.
    let draft = "";

    function box() { return sheet?.querySelector('textarea[name="said"]'); }

    function keepDraft() {
      const t = box();
      if (t) draft = t.value;
    }

    function restoreDraft() {
      const t = box();
      // Only when the server sent nothing back itself. A failed ask returns
      // the words it could not answer, and those are the newer ones.
      if (t && !t.value && draft) t.value = draft;
    }

    // Pulls /buddy and lifts its sheet out. The response is a whole page
    // because it has to be one for the scriptless path; taking one element out
    // of it is the cheapest possible upgrade.
    async function fetchSheet() {
      const res = await fetch("/buddy?from=" + encodeURIComponent(location.pathname),
        { headers: { "X-Requested-With": "fetch" } });
      if (!res.ok) return null;
      const doc = new DOMParser().parseFromString(await res.text(), "text/html");
      return doc.querySelector(".sheet");
    }

    function wireSheet() {
      restoreDraft();

      // Bound to the control itself as well as to the form's submit. Two
      // routes to the same call, because this is the only way out of a modal
      // sheet and it has failed in the field three times without failing once
      // in a test — belt and braces is the honest response to that.
      const shut = sheet.querySelector(".shut");
      shut?.addEventListener("click", e => {
        e.preventDefault();
        if (!close()) leave(shut.form);
      });

      // Enter sends, Shift+Enter is the newline — the same rule the slot has,
      // because it is the same interaction and because it is how a message is
      // sent in the room this product lives in.
      //
      // Bound here rather than with the slot's: that one is attached once at
      // load to the page's own box, and this box does not exist yet. Without
      // it, pressing return on a phone put a newline in and nothing happened,
      // which is what "I can't send a message to the coach" was.
      box()?.addEventListener("keydown", e => {
        if (e.key !== "Enter" || e.shiftKey) return;
        e.preventDefault();
        if (e.target.value.trim()) e.target.form.requestSubmit();
      });

      sheet.addEventListener("submit", async e => {
        const form = e.target;
        const action = form.getAttribute("action");
        // The capture slot and the timer are the rest of the product reached
        // from in here. Let them navigate: starting a timer is leaving the
        // conversation to go and do the thing, which is the point of it.
        if (action !== "/buddy/say" && action !== "/buddy/close") return;
        e.preventDefault();

        if (action === "/buddy/close") {
          // Not posted here: the dialog's own close event does it, so every
          // route out — Escape, the backdrop, this button — forgets the
          // conversation exactly once and in one place. Unless it did not
          // close, in which case the browser gets its form back.
          if (!close()) leave(form);
          return;
        }

        const data = formDataFor(form, e.submitter);
        keepDraft();

        // The network can go while the question is in the air, and this is the
        // screen you are on when you cannot start anything else — so it says
        // so, in the slot's own words and with the slot's own shape.
        //
        // It said nothing at all before. `post` and `fetchSheet` were awaited
        // bare, so a failure was an uncaught rejection: the button never
        // moved, the box never cleared, nothing appeared, and because there is
        // no spinner either, a slow answer and a dead one looked identical.
        const pressed = form.querySelector(".post");
        if (pressed) pressed.disabled = true;
        let fresh;
        try {
          const res = await post(action, data);
          // A 5xx is the same event as a dropped connection from here: the
          // words did not land. `fetch` only rejects on the network, so this
          // is the half that would otherwise pass for success.
          if (!res.ok) throw new Error(res.status);
          // Whatever was said is now in the window on the server, so the sheet
          // is re-read rather than patched. One source for what the
          // conversation is, and no chance of the two disagreeing.
          fresh = await fetchSheet();
        } catch {
          // The words are still in the box, which is why the box is never
          // cleared until the server has said something back.
          saidInSheet(form, "Not sent — Squirrel cannot reach Buddy. Your words are still here; try again in a moment.");
          if (pressed) pressed.disabled = false;
          return;
        }
        if (!fresh) { location.href = "/buddy"; return; }
        // The box empties when something was actually said, and keeps its
        // words when the server sent them back.
        draft = "";
        sheet.replaceWith(fresh);
        sheet = fresh;
        wireSheet();
        announce("Squirrel answered");
        sheet.querySelector(".talk")?.lastElementChild?.scrollIntoView({ block: "nearest" });
        box()?.focus();
      });
    }

    // What happened, on the sheet, in the slot's own voice. The element is in
    // the markup rather than made here so the scriptless path renders the same
    // shape and the stylesheet needs no second rule.
    function saidInSheet(form, words) {
      const said = form.querySelector("#saysaid");
      if (!said) return;
      said.textContent = words;
      said.className = "slotsaid bad";
      said.hidden = false;
      // No timeout. A failure you have to be quick to read is a failure you
      // will meet again without knowing why — the same reason the slot's own
      // bad news stays until something else happens.
    }

    // Posted the way a form posts, and that is not a detail.
    //
    // A FormData body goes out as multipart, and Go's ParseForm only reads
    // urlencoded — it does not error on multipart, it just finds nothing. So
    // every message sent from the sheet arrived at the server with no words in
    // it, was treated as "nothing was said", and redirected back. The box
    // cleared and nothing happened, which is what "I can't send a message to
    // the coach" was.
    //
    // The service worker's own capture flush has always done it this way. This
    // is the same shape, so the script's path and the scriptless path put
    // identical bytes on the wire.
    function post(action, data) {
      return fetch(action, {
        method: "POST",
        headers: { "Content-Type": "application/x-www-form-urlencoded" },
        body: new URLSearchParams(data),
      });
    }

    // FormData's second argument is what carries which button was pressed, and
    // it is newer than the rest of what this file uses — a browser without it
    // throws rather than ignoring it. That matters more here than it looks:
    // the submit has already been prevented by the time this runs, so a throw
    // means the form neither posts nor navigates and the press does nothing at
    // all. The chips would go the same way, since which chip is a submitter.
    function formDataFor(form, submitter) {
      try {
        return new FormData(form, submitter);
      } catch {
        const data = new FormData(form);
        if (submitter?.name) data.append(submitter.name, submitter.value);
        return data;
      }
    }

    let dialog = null;

    // Closes, and says whether it actually did.
    //
    // That return value is the whole of this. The submit handler used to
    // preventDefault and then call this, so if anything in here failed for any
    // reason the native post was already dead and the press did nothing at all
    // — a sheet you cannot get out of, on the one screen with no other way
    // back. Reported three times, and never once reproducible here.
    //
    // So the fallback is real rather than assumed: the caller checks, and
    // posts the form the browser's own way if the dialog is still standing.
    function close() {
      try {
        keepDraft();
        dialog?.close();
      } catch {
        // Whatever went wrong, the answer is the same one below.
      }
      return !dialog?.open;
    }

    // The last resort, and it must not go back through the submit handler that
    // just failed to work: form.submit() posts without firing submit at all,
    // so the server closes the conversation and redirects to where the acorn
    // was pressed. Slower than the dialog closing, and it always happens.
    function leave(form) {
      try {
        form?.submit();
      } catch {
        location.href = "/buddy/close";
      }
    }

    acorn.addEventListener("click", async e => {
      e.preventDefault();
      const fresh = await fetchSheet();
      // No sheet means no upgrade, so the link does what the link does. A
      // failure here must never leave the acorn doing nothing.
      if (!fresh) { location.href = acorn.href; return; }

      if (!dialog) {
        dialog = document.createElement("dialog");
        dialog.className = "coachsheet";
        document.body.appendChild(dialog);
        // Closing by any route — Escape, the backdrop, the button — is the
        // same close, and it forgets the conversation on the server exactly
        // once.
        dialog.addEventListener("close", () => {
          post("/buddy/close", new FormData());
        });
        dialog.addEventListener("click", e => {
          if (e.target === dialog) close();
        });
      }
      dialog.replaceChildren(fresh);
      sheet = fresh;
      wireSheet();
      // Buddy is reached from the lid's menu now, so the menu has to shut
      // behind you: a disclosure left standing open under a modal sheet is
      // waiting there when the sheet closes.
      acorn.closest("details.where")?.removeAttribute("open");
      dialog.showModal();
      box()?.focus();
    });
  })();

  // ---------------------------------------------------------------- //
  // The lid's two panels.
  //
  // A <details> is the right thing here — it opens with no script at all, and
  // it is the same grammar as the card's one question — but on its own it has
  // no idea it is a menu. Left alone it stays open until you press the summary
  // again, so the way out of it is a target you have to go back and find.
  //
  // Everything below is the part that makes it behave like a menu, and none of
  // it is load-bearing: with this file absent both panels still open, still
  // close on their own summary, and still work.
  // ---------------------------------------------------------------- //
  (() => {
    const panels = [...document.querySelectorAll("details.where, details.findbox")];
    if (!panels.length) return;

    function shut(except) {
      for (const p of panels) {
        if (p !== except && p.open) p.open = false;
      }
    }

    for (const panel of panels) {
      panel.addEventListener("toggle", () => {
        if (!panel.open) return;
        // One at a time. Two panels hanging off the same corner is two things
        // to close and one of them covering the other.
        shut(panel);
        // Asking for the field is asking to type in it.
        panel.querySelector('input[type="search"]')?.focus({ preventScroll: true });
      });
    }

    // Anywhere else means anywhere else, including the page behind the panel.
    // The summary is inside the details, so opening one never closes it.
    document.addEventListener("click", e => {
      for (const panel of panels) {
        if (panel.open && !panel.contains(e.target)) panel.open = false;
      }
    });

    // And the key that means "not this", which is what closes the sheet too.
    // Focus goes back to the control that opened it rather than being dropped
    // on the page, which is where a keyboard would otherwise have to start
    // again from.
    document.addEventListener("keydown", e => {
      if (e.key !== "Escape") return;
      const open = panels.find(p => p.open);
      if (!open) return;
      open.open = false;
      open.querySelector("summary")?.focus({ preventScroll: true });
    });
  })();
})();
