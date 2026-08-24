// The swap.
//
// JavaScript is required on this page — the progressive-enhancement rule was
// retired along with Principle 2 on 24 August 2026 — and this file is what
// makes a press feel like a press instead of a page load.
//
// It posts the same form to the same URL the browser would have. The only
// difference is one header, and what comes back is the same HTML the page would
// have rendered for those turns. There is no JSON here and no template; a
// second description of a card is how the two ends grow apart.
(() => {
  const thread = document.getElementById("thread");
  const say = document.getElementById("threadsay");
  if (!thread) return;

  // Anything this file changes without a navigation has to be said out loud
  // too, or the screen is silent for exactly the person who cannot see it
  // change.
  function announce(what) {
    if (say) say.textContent = what || "";
  }

  function toTheEnd() {
    const last = thread.lastElementChild;
    if (last && last.scrollIntoView) last.scrollIntoView({ block: "end", behavior: "smooth" });
  }

  // Controls belong to the live edge alone. When new turns arrive the turns
  // that were the edge stop being it — the same rule the server renders by,
  // applied to what is already on the screen. Without this a card keeps a
  // button that acts on a state nobody is looking at any more.
  function retire() {
    thread.querySelectorAll(".turn .turnacts, .turn .turnchips, .turn .faces")
      .forEach(el => el.remove());
  }

  // Which button was pressed, since a form with several buttons posts
  // different things depending on which one. The submit event does not carry
  // it everywhere, so it is caught on the way down.
  document.addEventListener("click", event => {
    const button = event.target.closest("button[type=submit], button:not([type])");
    if (button && button.form) button.form.__pressed = button;
  }, true);

  // Capture stays with pile.js — the camera, the stash and the offline path
  // all live there, and splitting the slot across two files would have to move
  // all of it. These are how it hands the answer back.
  window.__threadAppend = html => {
    retire();
    thread.insertAdjacentHTML("beforeend", html);
    toTheEnd();
    const last = thread.querySelector(".turn:last-child .bub");
    announce(last ? last.textContent : "");
  };
  window.__threadSay = what => {
    retire();
    thread.insertAdjacentHTML("beforeend",
      '<div class="turn frombuddy"><p class="bub"></p></div>');
    // textContent, not markup: this is the only place a word reaches the page
    // without having come back from the server, and it is written as text.
    thread.lastElementChild.querySelector(".bub").textContent = what;
    toTheEnd();
    announce(what);
  };

  document.addEventListener("submit", async event => {
    const form = event.target;
    if (!(form instanceof HTMLFormElement)) return;
    if (form.method.toLowerCase() !== "post") return;
    // The slot is pile.js's. See above.
    if (form.classList.contains("slot")) return;
    // A browser with no fetch: let it navigate, which still works.
    if (typeof fetch !== "function") return;

    event.preventDefault();

    const file = form.querySelector('input[type="file"]');
    const carrying = !!file?.files?.length;
    // The same bytes the form itself would have sent. FormData when there is a
    // photograph, because that is what multipart is for; URLSearchParams when
    // there is not, so the worker can still hold it offline.
    const data = new FormData(form);
    const pressed = form.__pressed;
    if (pressed && pressed.name) data.set(pressed.name, pressed.value);
    const body = carrying ? data
      : new URLSearchParams([...data].filter(([, v]) => typeof v === "string"));

    const init = {
      method: "POST", body, credentials: "same-origin",
      headers: { "X-Thread": "fragment" },
    };
    if (!carrying) init.headers["Content-Type"] = "application/x-www-form-urlencoded";

    const post = form.querySelector(".post");
    if (post) post.disabled = true;
    try {
      const res = await fetch(form.action, init);
      if (!res.ok) throw new Error("the press did not land");
      const html = await res.text();
      // Nothing back means the server decided there was nothing to do — an
      // empty box, pressed. Say nothing; it did nothing.
      if (html.trim()) {
        retire();
        // Parsed as HTML deliberately, and it is safe for one reason worth
        // stating: this is our own markup from our own origin, rendered by
        // html/template, which escapes everything a person typed. That is not
        // an assumption — TestTheSlotEscapesWhatItGivesBack fails if a turn's
        // words ever reach the page unescaped. Nothing here ever writes text
        // that did not come back from the server.
        thread.insertAdjacentHTML("beforeend", html);
        const box = form.querySelector("textarea");
        if (box) { box.value = ""; box.style.height = "auto"; }
        if (file && file.files?.length) file.value = "";
        toTheEnd();
        const last = thread.querySelector(".turn:last-child .bub");
        announce(last ? last.textContent : "");
      }
    } catch {
      // The press did not land, and it must not look like one that did. The
      // form goes through the ordinary way, which shows whatever the server
      // actually says — and the words are still in the box until it does.
      form.submit();
    } finally {
      if (post) post.disabled = false;
      form.__pressed = null;
    }
  });

  // Letters are actions, on the note Buddy is holding out.
  //
  // The deck's own keys came with a machine for stamping a card and holding it
  // still so an undo had somewhere to be. None of that crosses: the answer here
  // is a new turn, and the way back travels with it. What crosses is the letters
  // — d, k, x, t — because typing them is how triage is done at a desk.
  const PILE_KEYS = { d: "done", k: "keep", x: "drop", t: "task" };

  document.addEventListener("keydown", event => {
    if (event.metaKey || event.ctrlKey || event.altKey) return;
    // A field owns its own letters.
    const on = event.target;
    if (on && (on.tagName === "TEXTAREA" || on.tagName === "INPUT" || on.isContentEditable)) return;

    const act = PILE_KEYS[event.key.toLowerCase()];
    if (!act) return;
    // The live edge only, and only when it is holding a note out: the same
    // rule the server renders by.
    const card = thread.querySelector(".turn:last-child .turncard");
    const press = card?.querySelector(`input[name="act"][value="${act}"]`)?.form
      ?.querySelector("button");
    if (!press) return;
    event.preventDefault();
    press.click();
  });

  // Open at the end of the conversation. Without this the page opens at the
  // top, which is the beginning of everything you have ever said.
  toTheEnd();
})();
