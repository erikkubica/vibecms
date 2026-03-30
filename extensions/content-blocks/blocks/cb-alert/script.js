(function () {
  document.querySelectorAll('[data-block="cb-alert"]').forEach(function (block, index) {
    var dismissible = block.getAttribute("data-dismissible") === "true";
    if (!dismissible) return;

    var storageKey = "vb-cb-alert-dismissed-" + index + "-" + window.location.pathname;
    var closeBtn = block.querySelector(".vb-cb-alert__close");

    /* Check if previously dismissed */
    try {
      if (localStorage.getItem(storageKey) === "1") {
        block.style.display = "none";
        return;
      }
    } catch (e) {
      /* localStorage not available */
    }

    if (!closeBtn) return;

    closeBtn.addEventListener("click", function () {
      block.classList.add("vb-cb-alert--dismissed");

      setTimeout(function () {
        block.style.display = "none";
      }, 300);

      try {
        localStorage.setItem(storageKey, "1");
      } catch (e) {
        /* localStorage not available */
      }
    });
  });

  // Load Lucide icons
  if (!window.lucide && !document.querySelector('script[data-lucide-loader]')) {
    var s = document.createElement('script');
    s.src = 'https://unpkg.com/lucide@0.460.0/dist/umd/lucide.min.js';
    s.setAttribute('data-lucide-loader', 'true');
    s.onload = function() { if (window.lucide) window.lucide.createIcons(); };
    document.head.appendChild(s);
  } else if (window.lucide) {
    window.lucide.createIcons();
  }
})();
