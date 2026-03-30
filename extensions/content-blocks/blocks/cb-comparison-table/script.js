(function () {
  document.querySelectorAll('[data-block="cb-comparison-table"]').forEach(function (block) {
    block.querySelectorAll('.vb-comparison-table__value-cell').forEach(function (cell) {
      var val = (cell.getAttribute('data-value') || '').trim().toLowerCase();
      if (val === 'yes') {
        cell.textContent = '';
        var span = document.createElement('span');
        span.className = 'vb-comparison-table__check';
        span.setAttribute('aria-label', 'Yes');
        span.textContent = '\u2713';
        cell.appendChild(span);
      } else if (val === 'no') {
        cell.textContent = '';
        var span = document.createElement('span');
        span.className = 'vb-comparison-table__cross';
        span.setAttribute('aria-label', 'No');
        span.textContent = '\u2717';
        cell.appendChild(span);
      }
    });
  });
})();
