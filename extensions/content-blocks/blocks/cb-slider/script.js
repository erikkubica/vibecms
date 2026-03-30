(function () {
  document.querySelectorAll('[data-block="cb-slider"]').forEach(function (slider) {
    var slides = slider.querySelectorAll(".vb-cb-slider__slide");
    var dots = slider.querySelectorAll(".vb-cb-slider__dot");
    var prevBtn = slider.querySelector(".vb-cb-slider__arrow--prev");
    var nextBtn = slider.querySelector(".vb-cb-slider__arrow--next");
    var autoplay = slider.getAttribute("data-autoplay") === "true";
    var current = 0;
    var total = slides.length;
    var interval = null;

    if (total <= 1) {
      if (prevBtn) prevBtn.style.display = "none";
      if (nextBtn) nextBtn.style.display = "none";
      return;
    }

    function goTo(index) {
      slides[current].classList.remove("vb-cb-slider__slide--active");
      dots[current].classList.remove("vb-cb-slider__dot--active");
      current = (index + total) % total;
      slides[current].classList.add("vb-cb-slider__slide--active");
      dots[current].classList.add("vb-cb-slider__dot--active");
    }

    function next() {
      goTo(current + 1);
    }

    function prev() {
      goTo(current - 1);
    }

    function startAutoplay() {
      if (autoplay && !interval) {
        interval = setInterval(next, 5000);
      }
    }

    function stopAutoplay() {
      if (interval) {
        clearInterval(interval);
        interval = null;
      }
    }

    nextBtn.addEventListener("click", function () {
      stopAutoplay();
      next();
      startAutoplay();
    });

    prevBtn.addEventListener("click", function () {
      stopAutoplay();
      prev();
      startAutoplay();
    });

    dots.forEach(function (dot) {
      dot.addEventListener("click", function () {
        stopAutoplay();
        goTo(parseInt(dot.getAttribute("data-slide"), 10));
        startAutoplay();
      });
    });

    slider.addEventListener("mouseenter", stopAutoplay);
    slider.addEventListener("mouseleave", startAutoplay);

    startAutoplay();
  });
})();
