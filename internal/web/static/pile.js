// Progressive enhancement, and nothing here is load-bearing. Every action on
// this page is a form submission or a link that works with this file absent;
// what this adds is the stamp, the moment the card holds still so the undo has
// somewhere to be, one key per action, and search that answers as you type.
(() => {
  const find = document.querySelector(".find input");
  const stage = document.getElementById("stage");

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

      setTimeout(() => {
        card.classList.add("leaving");
        setTimeout(() => {
          // Hand the real submission back to the browser. The server does the
          // write and answers 303, so a reload never repeats it.
          button.form.requestSubmit(button);
        }, 440);
      }, 1150);
    }

    // Choosing an interval takes the place of the actions rather than sitting
    // under them, the way the comp draws it. The disclosure already does the
    // showing; the class does the hiding, and neither exists without this file.
    function choosing(open) {
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

  const KEYS = { d: "done", k: "keep", x: "drop" };

  function skip() {
    const card = deck?.card;
    if (!card?.dataset.skip) return;
    const url = new URL(location.href);
    url.searchParams.set("after", card.dataset.skip);
    url.searchParams.delete("undo");
    url.searchParams.delete("was");
    url.searchParams.delete("state");
    location.assign(url);
  }

  addEventListener("keydown", e => {
    if (e.target === find) {
      if (e.key === "Escape") { find.value = ""; clearTimeout(timer); swap(""); }
      return;
    }
    if (e.key === "/") { e.preventDefault(); find?.focus(); return; }
    // A focused control owns space and enter; that is the platform's contract.
    if ((e.key === " " || e.key === "Enter") && e.target.closest("button, summary, a")) return;
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
})();
