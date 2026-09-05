(function () {
  "use strict";

  var HOLD = 1150;
  var TRAVEL = 260;

  var still = window.matchMedia("(prefers-reduced-motion: reduce)").matches;
  var byPress = window.matchMedia("(hover: none) and (pointer: coarse)").matches;

  var CHEVRON =
    '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.6" ' +
    'stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M7 10l5 5 5-5"/></svg>';

  // Letters act, arrows move. The board draws a key on every stamp, and until
  // now they were a promise: the letters were on the strips and nothing read
  // them. Focus is the platform's own selection model — the strip you are in —
  // which is what the chores screen used before the board existed.
  var strips = function () {
    return [].slice.call(document.querySelectorAll(".strip.answerable"));
  };

  var focused = function () {
    return document.activeElement && document.activeElement.closest(".strip.answerable");
  };

  var openerIn = function (strip) {
    return strip.querySelector(".opener");
  };

  var shut = function (strip) {
    strip.classList.remove("open");
    var opener = openerIn(strip);
    if (opener) opener.setAttribute("aria-expanded", "false");
  };

  var show = function (strip) {
    strips().forEach(function (other) {
      if (other !== strip) shut(other);
    });
    strip.classList.add("open");
    var opener = openerIn(strip);
    if (opener) opener.setAttribute("aria-expanded", "true");
  };

  var focus = function (strip) {
    if (!strip) return;
    if (byPress) {
      show(strip);
      var opener = openerIn(strip);
      if (opener) opener.focus();
      return;
    }
    var first = strip.querySelector(".stamp");
    if (first) first.focus();
  };

  if (byPress) {
    document.documentElement.classList.add("presses");
    window.requestAnimationFrame(function () {
      window.requestAnimationFrame(function () {
        document.documentElement.classList.add("eased");
      });
    });
    strips().forEach(function (strip) {
      var opener = document.createElement("button");
      opener.type = "button";
      opener.className = "opener";
      opener.setAttribute("aria-expanded", "false");
      opener.setAttribute("aria-label", "what you can do with this");
      opener.innerHTML = CHEVRON;
      strip.appendChild(opener);
    });

    document.addEventListener("click", function (e) {
      var strip = e.target.closest(".strip.answerable");
      if (!strip) return;
      if (e.target.closest("a, .stamp, input, label, textarea")) return;
      e.preventDefault();
      if (strip.classList.contains("open")) {
        shut(strip);
        return;
      }
      show(strip);
    });

    document.addEventListener("keydown", function (e) {
      if (e.key !== "Escape") return;
      var open = document.querySelector(".strip.answerable.open");
      if (!open) return;
      e.preventDefault();
      shut(open);
      var opener = openerIn(open);
      if (opener) opener.focus();
    });
  }

  document.addEventListener("keydown", function (e) {
    if (e.metaKey || e.ctrlKey || e.altKey) return;
    var typing = e.target.matches("input, textarea, select");
    if (typing) return;

    var all = strips();
    if (!all.length) return;
    var here = focused();

    if (e.key === "ArrowDown" || e.key === "ArrowUp") {
      e.preventDefault();
      var at = here ? all.indexOf(here) : -1;
      var next = e.key === "ArrowDown" ? at + 1 : at - 1;
      if (next < 0) next = 0;
      if (next > all.length - 1) next = all.length - 1;
      focus(all[next]);
      return;
    }

    if (e.key.length !== 1) return;
    if (!here) return;
    var letter = e.key.toUpperCase();
    if (byPress) show(here);
    var stamps = [].slice.call(here.querySelectorAll(".stamp"));
    for (var i = 0; i < stamps.length; i++) {
      var key = stamps[i].querySelector(".k");
      if (key && key.textContent.trim().toUpperCase() === letter) {
        e.preventDefault();
        stamps[i].click();
        return;
      }
    }
  });

  var announce = function (text) {
    var region = document.getElementById("announce");
    if (region) region.textContent = text;
  };

  var pending = null;

  document.addEventListener("submit", function (e) {
    var form = e.target;
    if (!form.classList.contains("stamps")) return;
    if (e.submitter && e.submitter.getAttribute("formmethod") === "get") return;
    var strip = form.closest(".strip");
    if (!strip) return;

    e.preventDefault();

    if (pending && pending.strip === strip) {
      pending.timers.forEach(window.clearTimeout);
      pending = null;
    }

    var pressed = e.submitter;
    strip.classList.add("struck");

    var label = pressed
      ? [].filter
          .call(pressed.childNodes, function (n) {
            return n.nodeType === 3;
          })
          .map(function (n) {
            return n.textContent;
          })
          .join("")
          .trim()
      : "";
    announce(label ? label + " — noted, sending" : "noted, sending");

    var timers = [];
    pending = { strip: strip, timers: timers, label: label };

    var go = function () {
      pending = null;
      if (pressed && pressed.name) {
        var carried = document.createElement("input");
        carried.type = "hidden";
        carried.name = pressed.name;
        carried.value = pressed.value;
        form.appendChild(carried);
      }
      form.submit();
    };

    if (still) {
      timers.push(window.setTimeout(go, HOLD));
      return;
    }
    timers.push(
      window.setTimeout(function () {
        strip.classList.add("leaving");
        timers.push(window.setTimeout(go, TRAVEL));
      }, HOLD)
    );
  });

  document.addEventListener("keydown", function (e) {
    if (e.key !== "Escape" || !pending) return;
    e.preventDefault();
    pending.timers.forEach(window.clearTimeout);
    pending.strip.classList.remove("struck", "leaving");
    announce(pending.label ? pending.label + " — cancelled" : "cancelled");
    pending = null;
  });
})();
