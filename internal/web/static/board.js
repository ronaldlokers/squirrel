(function () {
  "use strict";

  var HOLD = 1150;
  var TRAVEL = 260;

  var still = window.matchMedia("(prefers-reduced-motion: reduce)").matches;

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

  var focus = function (strip) {
    if (!strip) return;
    var first = strip.querySelector(".stamp");
    if (first) first.focus();
  };

  document.addEventListener("keydown", function (e) {
    if (e.metaKey || e.ctrlKey || e.altKey) return;
    var typing = e.target.matches("input, textarea");
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
    var letter = e.key.toUpperCase();
    // Nothing focused: the first press says where you are rather than acting
    // on something you did not choose.
    if (!here) {
      var carries = all.filter(function (strip) {
        return [].some.call(strip.querySelectorAll(".stamp .k"), function (k) {
          return k.textContent.trim().toUpperCase() === letter;
        });
      });
      if (carries.length) {
        e.preventDefault();
        focus(carries[0]);
      }
      return;
    }
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

  document.addEventListener("submit", function (e) {
    var form = e.target;
    if (!form.classList.contains("stamps")) return;
    var strip = form.closest(".strip");
    if (!strip) return;

    e.preventDefault();
    var pressed = e.submitter;
    strip.classList.add("struck");

    var go = function () {
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
      window.setTimeout(go, HOLD);
      return;
    }
    window.setTimeout(function () {
      strip.classList.add("leaving");
      window.setTimeout(go, TRAVEL);
    }, HOLD);
  });
})();
