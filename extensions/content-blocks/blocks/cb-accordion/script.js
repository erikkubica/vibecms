(function () {
  document.querySelectorAll('[data-block="cb-accordion"]').forEach(function (block) {
    var allowMultiple = block.getAttribute("data-allow-multiple") === "true";
    var headers = block.querySelectorAll(".vb-cb-accordion__header");

    headers.forEach(function (header) {
      header.addEventListener("click", function () {
        var item = header.closest(".vb-cb-accordion__item");
        var panel = item.querySelector(".vb-cb-accordion__panel");
        var isOpen = item.classList.contains("vb-cb-accordion__item--open");

        if (!allowMultiple && !isOpen) {
          block.querySelectorAll(".vb-cb-accordion__item--open").forEach(function (openItem) {
            var openPanel = openItem.querySelector(".vb-cb-accordion__panel");
            openItem.classList.remove("vb-cb-accordion__item--open");
            openItem.querySelector(".vb-cb-accordion__header").setAttribute("aria-expanded", "false");
            openPanel.style.maxHeight = null;
          });
        }

        if (isOpen) {
          item.classList.remove("vb-cb-accordion__item--open");
          header.setAttribute("aria-expanded", "false");
          panel.style.maxHeight = null;
        } else {
          item.classList.add("vb-cb-accordion__item--open");
          header.setAttribute("aria-expanded", "true");
          panel.style.maxHeight = panel.scrollHeight + "px";
        }
      });
    });
  });
})();
