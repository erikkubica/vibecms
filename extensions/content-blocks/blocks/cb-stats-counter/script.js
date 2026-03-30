(function () {
  "use strict";

  function animateCounter(el) {
    var target = parseFloat(el.getAttribute("data-target"));
    if (isNaN(target)) return;

    var isDecimal = String(target).indexOf(".") !== -1;
    var decimalPlaces = isDecimal ? String(target).split(".")[1].length : 0;
    var duration = 1800;
    var startTime = null;

    function easeOutCubic(t) {
      return 1 - Math.pow(1 - t, 3);
    }

    function step(timestamp) {
      if (!startTime) startTime = timestamp;
      var progress = Math.min((timestamp - startTime) / duration, 1);
      var easedProgress = easeOutCubic(progress);
      var current = easedProgress * target;

      if (isDecimal) {
        el.textContent = current.toFixed(decimalPlaces);
      } else {
        el.textContent = Math.floor(current);
      }

      if (progress < 1) {
        requestAnimationFrame(step);
      } else {
        el.textContent = isDecimal ? target.toFixed(decimalPlaces) : target;
      }
    }

    requestAnimationFrame(step);
  }

  function initCounters() {
    var blocks = document.querySelectorAll('[data-block="cb-stats-counter"]');

    if (!blocks.length) return;

    var observer = new IntersectionObserver(
      function (entries) {
        entries.forEach(function (entry) {
          if (entry.isIntersecting) {
            var values = entry.target.querySelectorAll(
              ".vb-stats-counter__value"
            );
            values.forEach(function (el) {
              if (!el.dataset.animated) {
                el.dataset.animated = "true";
                animateCounter(el);
              }
            });
            observer.unobserve(entry.target);
          }
        });
      },
      { threshold: 0.2 }
    );

    blocks.forEach(function (block) {
      observer.observe(block);
    });
  }

  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", initCounters);
  } else {
    initCounters();
  }
})();
