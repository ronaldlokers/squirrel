(function () {
  "use strict";

  var HOLD = 1150;
  var TRAVEL = 260;

  var still = window.matchMedia("(prefers-reduced-motion: reduce)").matches;

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
