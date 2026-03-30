(function () {
  document.querySelectorAll('[data-block="cb-testimonials"]').forEach(function (block) {
    var track = block.querySelector('.vb-testimonials__track');
    var slides = block.querySelectorAll('.vb-testimonials__slide');
    var prevBtn = block.querySelector('.vb-testimonials__arrow--prev');
    var nextBtn = block.querySelector('.vb-testimonials__arrow--next');
    var dotsContainer = block.querySelector('.vb-testimonials__dots');
    var current = 0;
    var total = slides.length;

    if (total === 0) return;

    // Initialize star ratings
    block.querySelectorAll('.vb-testimonials__stars').forEach(function (stars) {
      var rating = parseInt(stars.getAttribute('data-rating') || '5', 10);
      var starEls = stars.querySelectorAll('.vb-testimonials__star');
      for (var i = 0; i < starEls.length; i++) {
        if (i < rating) {
          starEls[i].classList.add('vb-testimonials__star--filled');
        }
      }
    });

    // Build dots
    for (var i = 0; i < total; i++) {
      var dot = document.createElement('button');
      dot.className = 'vb-testimonials__dot' + (i === 0 ? ' vb-testimonials__dot--active' : '');
      dot.setAttribute('aria-label', 'Go to testimonial ' + (i + 1));
      dot.setAttribute('data-index', String(i));
      dotsContainer.appendChild(dot);
    }

    function goTo(index) {
      if (index < 0) index = total - 1;
      if (index >= total) index = 0;
      current = index;

      for (var i = 0; i < slides.length; i++) {
        slides[i].style.transform = 'translateX(' + (-current * 100) + '%)';
      }

      var dots = dotsContainer.querySelectorAll('.vb-testimonials__dot');
      for (var i = 0; i < dots.length; i++) {
        dots[i].classList.toggle('vb-testimonials__dot--active', i === current);
      }
    }

    prevBtn.addEventListener('click', function () { goTo(current - 1); });
    nextBtn.addEventListener('click', function () { goTo(current + 1); });
    dotsContainer.addEventListener('click', function (e) {
      var idx = e.target.getAttribute('data-index');
      if (idx !== null) goTo(parseInt(idx, 10));
    });
  });
})();
