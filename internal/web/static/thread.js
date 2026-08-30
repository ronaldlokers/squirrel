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

  // The dock used to be fixed to the bottom and cover whatever was under it,
  // so this measured its height on every resize and reserved that much padding
  // through a --dockspace custom property. The dock is the last row of the
  // thread's own grid now, so there is nothing to reserve and nothing to
  // measure: a slot grown to four lines shortens the scroll region by exactly
  // its own growth, which is what a layout does and a ResizeObserver was
  // standing in for.

  function toTheEnd() {
    const last = thread.lastElementChild;
    if (last && last.scrollIntoView) last.scrollIntoView({ block: "end", behavior: "smooth" });
    drawKeys();
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
    // `.said` as well as `.bub`, and this is not a tidy-up.
    //
    // Buddy's words stopped being a bubble on 26 August 2026 and this line
    // announces the newest turn. Left reading `.bub` alone it found nothing
    // for every turn Buddy appends, so the live region said nothing — the
    // screen would have gone silent for exactly the person who cannot see it
    // change, which is the failure this whole function exists to prevent.
    const last = thread.querySelector(".turn:last-child .bub, .turn:last-child .said");
    announce(last ? last.textContent : "");
  };
  window.__threadSay = what => {
    retire();
    thread.insertAdjacentHTML("beforeend",
      '<div class="turn frombuddy"><p class="said"></p></div>');
    // textContent, not markup: this is the only place a word reaches the page
    // without having come back from the server, and it is written as text.
    thread.lastElementChild.querySelector(".said").textContent = what;
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
      // A redirect is the server saying this press is a navigation rather than
      // a turn. fetch follows one without telling anybody, so what arrives is
      // a whole page — and pasting that into the room it came from is exactly
      // the bug it looked like: a room, and its navigation, inside the room.
      // Going where it points is what the answer meant.
      if (res.redirected) {
        window.location.assign(res.url);
        return;
      }
      const html = await res.text();
      // An answer that re-draws something already on the screen rather than
      // adding to it — turning the calendar's month is the one that does this.
      // It comes back under the id it already had; swapping it keeps the
      // conversation the length it was, which is what paging should cost.
      const instead = res.headers.get("X-Replaces");
      if (instead && html.trim()) {
        const there = document.getElementById(instead);
        if (there) {
          there.outerHTML = html;
          const now = document.getElementById(instead);
          announce(now?.querySelector(".calhead b")?.textContent || "");
          return;
        }
      }
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
        const last = thread.querySelector(".turn:last-child .bub, .turn:last-child .said");
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

  function cap(legend) {
    const el = document.createElement("span");
    el.className = "key";
    el.setAttribute("aria-hidden", "true");
    el.textContent = legend;
    return el;
  }

  function pickerLegend(radio, row) {
    if (radio.name === "count") {
      return /^[0-9]$/.test(radio.value) ? radio.value : null;
    }
    if (radio.name !== "unit") return null;
    const first = radio.value[0]?.toLowerCase();
    if (!first) return null;
    const sharing = row.filter(other =>
      other.name === "unit" && other.value[0]?.toLowerCase() === first);
    return sharing.length === 1 ? first.toUpperCase() : null;
  }

  function drawKeys() {
    thread.querySelectorAll(".key").forEach(el => el.remove());
    const last = thread.querySelector(".turn:last-child");
    if (!last) return;

    for (const [letter, act] of Object.entries(PILE_KEYS)) {
      last.querySelector(`input[name="act"][value="${act}"]`)?.form
        ?.querySelector("button")?.append(cap(letter.toUpperCase()));
    }

    const pick = last.querySelector(".pick");
    if (!pick) return;
    const radios = [...pick.querySelectorAll('.pickrow input[type="radio"]')];
    for (const radio of radios) {
      const legend = pickerLegend(radio, radios);
      if (legend) radio.closest(".chip")?.append(cap(legend));
    }
    pick.querySelector(".make")?.append(cap("\u21b5"));
  }

  // A question in progress owns the keys.
  //
  // The deck earned this carve-out: 1-4 answered its interval question, and a
  // letter aimed at the picker must not act on the note behind it. The question
  // is a turn of its own now, so the rule is about the picker wherever it is —
  // and the digits came back with it, which the deck's own set lost when the
  // disclosure went. Roadmap v0.24.0 recorded that as worth restoring.
  //
  // The number row by digit, the unit row by first letter, and Enter answers.
  // Only the interval question: a digit inside the day picker would be a day of
  // the month, and guessing which of the two you meant would book the wrong one.
  function askedAKey(event) {
    const pick = thread.querySelector(".turn:last-child .pick");
    if (!pick) return false;

    if (event.key === "Enter") {
      pick.querySelector(".make")?.click();
      return true;
    }
    // A digit is a count; a letter is a unit, by the word's own first letter.
    // Both are radios, so choosing one is pressing its label.
    const want = /^[0-9]$/.test(event.key)
      ? `input[name="count"][value="${event.key}"]`
      : `input[name="unit"][value^="${event.key.toLowerCase()}"]`;
    const answer = pick.querySelector(want);
    if (!answer) return false;
    answer.checked = true;
    return true;
  }

  document.addEventListener("keydown", event => {
    if (event.metaKey || event.ctrlKey || event.altKey) return;
    // A field owns its own letters.
    const on = event.target;
    if (on && (on.tagName === "TEXTAREA" || on.tagName === "INPUT" || on.isContentEditable)) return;

    if (askedAKey(event)) {
      event.preventDefault();
      return;
    }

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

// The three times fill the field rather than being three of four answers.
//
// Delegated, because a calendar arrives in a fragment long after this file
// ran. Enhancement only: with the script gone the chips do nothing and the
// field still takes any time, which is the whole answer either way.
document.addEventListener("click", event => {
  const chip = event.target.closest?.("[data-at]");
  if (!chip) return;
  const field = chip.closest(".pickrow")?.querySelector(".attime");
  if (!field) return;
  field.value = chip.dataset.at;
  field.dispatchEvent(new Event("input", { bubbles: true }));
});
