(function () {
  document.querySelectorAll('[data-block="cb-reviews"]').forEach(function (block) {
    block.querySelectorAll('.vb-reviews__stars').forEach(function (stars) {
      var rating = parseInt(stars.getAttribute('data-rating') || '5', 10);
      var starEls = stars.querySelectorAll('.vb-reviews__star');
      for (var i = 0; i < starEls.length; i++) {
        if (i < rating) {
          starEls[i].classList.add('vb-reviews__star--filled');
        }
      }
    });
  });
})();
