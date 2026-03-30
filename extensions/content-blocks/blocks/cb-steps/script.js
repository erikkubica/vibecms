(function() {
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
