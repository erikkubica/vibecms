(function () {
  document.querySelectorAll('[data-block="cb-before-after"]').forEach(function (block) {
    var wrapper = block.querySelector(".vb-cb-before-after__wrapper");
    if (!wrapper) return;

    var beforeLayer = wrapper.querySelector(".vb-cb-before-after__before");
    var handle = wrapper.querySelector(".vb-cb-before-after__handle");
    var dragging = false;

    function getPosition(e) {
      var rect = wrapper.getBoundingClientRect();
      var clientX = e.touches ? e.touches[0].clientX : e.clientX;
      var x = clientX - rect.left;
      return Math.max(0, Math.min(x / rect.width, 1));
    }

    function update(ratio) {
      var pct = (ratio * 100).toFixed(2) + "%";
      beforeLayer.style.clipPath = "inset(0 " + (100 - ratio * 100).toFixed(2) + "% 0 0)";
      handle.style.left = pct;
    }

    function onStart(e) {
      e.preventDefault();
      dragging = true;
      update(getPosition(e));
    }

    function onMove(e) {
      if (!dragging) return;
      e.preventDefault();
      update(getPosition(e));
    }

    function onEnd() {
      dragging = false;
    }

    // Mouse events
    wrapper.addEventListener("mousedown", onStart);
    document.addEventListener("mousemove", onMove);
    document.addEventListener("mouseup", onEnd);

    // Touch events
    wrapper.addEventListener("touchstart", onStart, { passive: false });
    document.addEventListener("touchmove", onMove, { passive: false });
    document.addEventListener("touchend", onEnd);
  });
})();
