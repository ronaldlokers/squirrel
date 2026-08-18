// Progressive enhancement, and nothing here is load-bearing. Every action on
// this page is a form submission that works with this file absent; what this
// adds is the stamp, the moment the card holds still so the undo has somewhere
// to be, and one key per action.
(() => {
  const card = document.getElementById("card");
  const find = document.querySelector(".find input");

  // "/" focuses the search field on every page, including the empty pile and
  // the results list, where there is no card to act on.
  addEventListener("keydown", e => {
    if (e.key === "/" && e.target !== find) { e.preventDefault(); find.focus(); }
  });

  if (!card) return;                      // the empty pile, or a search
  const form = document.getElementById("actions");
  const stamp = document.getElementById("stamp");
  const stampText = document.getElementById("stampText");
  const undoRow = document.getElementById("undoRow");
  const said = document.getElementById("said");
  const every = form.querySelector("details.everyFallback");
  const chips = [...form.querySelectorAll("button[name=every]")];

  const STATES = {
    done:  { word: "DONE",    said: "marked done" },
    keep:  { word: "KEPT",    said: "kept as reference" },
    drop:  { word: "DROPPED", said: "dropped" },
    chore: { word: "CHORE",   said: "now a chore" }
  };

  // Habituation is the enemy: the stack never sits the same way twice.
  document.querySelectorAll(".behind").forEach((el, n) => {
    const rot = (n % 2 ? -1 : 1) * (0.7 + Math.random() * 1.3);
    const step = 1 + n * 0.8 + Math.random() * 0.5;
    el.style.transform =
      `translate(calc(var(--o) * ${step.toFixed(2)}), calc(var(--o) * ${(step * 1.1).toFixed(2)})) rotate(${rot.toFixed(2)}deg)`;
  });

  let going = false;

  // Act, then hold, then submit. The delay is not decoration: it is the moment
  // the spec asks for, and the undo has to be reachable while the card that it
  // undoes is still on the screen.
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
    document.getElementById("undo").focus({ preventScroll: true });

    setTimeout(() => {
      card.classList.add("leaving");
      setTimeout(() => {
        // Hand the real submission back to the browser. The server does the
        // write and answers 303, so a reload never repeats it.
        button.form.requestSubmit(button);
      }, 440);
    }, 1150);
  }

  form.addEventListener("click", e => {
    const b = e.target.closest("button[name=act], button[name=every]");
    if (!b || going) return;
    e.preventDefault();
    go(b);
  });

  // PUT IT BACK is the transition back to open, posted like any other.
  document.getElementById("undo").addEventListener("click", () => {
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

  // Letters are actions, always. Movement is space and the arrows, because in a
  // one-card topology j/k has nothing to move between and k is keep.
  const KEYS = { d: "done", k: "keep", x: "drop" };
  addEventListener("keydown", e => {
    if (e.target === find) {
      if (e.key === "Escape") { find.value = ""; find.form.submit(); }
      return;
    }
    // A focused control owns space and enter; that is the platform's contract.
    if ((e.key === " " || e.key === "Enter") && e.target.closest("button, summary")) return;
    if (going) return;

    // The chore interval is a question, not an action: C asks it, 1-4 answer
    // it, and ESC withdraws it. Nothing here fires a write on its own.
    if (every && every.open) {
      const n = "1234".indexOf(e.key);
      if (n >= 0 && chips[n]) { e.preventDefault(); go(chips[n]); return; }
      if (e.key === "Escape") { e.preventDefault(); every.open = false; }
      return;
    }
    if (e.key.toLowerCase() === "c" && every) {
      e.preventDefault();
      every.open = true;
      chips[0]?.focus({ preventScroll: true });
      return;
    }

    const act = KEYS[e.key.toLowerCase()];
    if (act) {
      const b = form.querySelector(`[data-act="${act}"]`);
      if (b) { e.preventDefault(); go(b); }
    }
  });
})();
