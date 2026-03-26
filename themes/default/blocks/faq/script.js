/**
 * FAQ Accordion Block
 *
 * This block relies on Alpine.js (x-data, x-show, x-collapse) for accordion
 * behavior. Alpine.js should be loaded globally by the theme or layout.
 *
 * If the x-collapse plugin is not available, this script provides a minimal
 * fallback that toggles visibility via max-height transitions.
 */
(function () {
  if (window.Alpine && window.Alpine.directive) {
    // Alpine.js is loaded — check if x-collapse plugin is available
    try {
      // If Alpine Collapse plugin is registered, nothing else needed
      if (Alpine.bound || document.querySelector('[x-collapse]')) {
        return;
      }
    } catch (e) {
      // Silently continue to fallback
    }
  }

  // Minimal fallback: if Alpine is not present, wire up plain JS toggles
  document.addEventListener('DOMContentLoaded', function () {
    if (window.Alpine) return; // Alpine will handle it

    document.querySelectorAll('.vb-faq-item').forEach(function (item) {
      var button = item.querySelector('.vb-faq-question');
      var answer = item.querySelector('.vb-faq-answer');
      var chevron = item.querySelector('.vb-faq-chevron');

      if (!button || !answer) return;

      answer.style.display = 'none';

      button.addEventListener('click', function () {
        var isOpen = answer.style.display !== 'none';
        answer.style.display = isOpen ? 'none' : 'block';
        if (chevron) {
          chevron.classList.toggle('rotate-180', !isOpen);
        }
        button.setAttribute('aria-expanded', !isOpen);
      });
    });
  });
})();
