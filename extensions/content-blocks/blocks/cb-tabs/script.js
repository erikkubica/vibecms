(function () {
  document.querySelectorAll('[data-block="cb-tabs"]').forEach(function (block) {
    var tabs = block.querySelectorAll(".vb-cb-tabs__tab");
    var panels = block.querySelectorAll(".vb-cb-tabs__panel");

    tabs.forEach(function (tab) {
      tab.addEventListener("click", function () {
        var index = tab.getAttribute("data-tab-index");

        tabs.forEach(function (t) {
          t.classList.remove("vb-cb-tabs__tab--active");
          t.setAttribute("aria-selected", "false");
        });

        panels.forEach(function (p) {
          p.classList.remove("vb-cb-tabs__panel--active");
        });

        tab.classList.add("vb-cb-tabs__tab--active");
        tab.setAttribute("aria-selected", "true");

        var activePanel = block.querySelector('[data-panel-index="' + index + '"]');
        if (activePanel) {
          activePanel.classList.add("vb-cb-tabs__panel--active");
        }
      });
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
