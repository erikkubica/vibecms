(function () {
  document.querySelectorAll('[data-block="cb-subscription-plans"]').forEach(function (block) {
    var toggle = block.querySelector(".vb-cb-subscription-plans__toggle");
    var monthlyLabel = block.querySelector(".vb-cb-subscription-plans__toggle-label--monthly");
    var yearlyLabel = block.querySelector(".vb-cb-subscription-plans__toggle-label--yearly");
    var amounts = block.querySelectorAll(".vb-cb-subscription-plans__amount");
    var periods = block.querySelectorAll(".vb-cb-subscription-plans__period");
    var isYearly = false;

    function updatePrices() {
      var key = isYearly ? "yearly" : "monthly";

      amounts.forEach(function (el) {
        el.textContent = el.getAttribute("data-" + key);
      });

      periods.forEach(function (el) {
        el.textContent = el.getAttribute("data-" + key);
      });

      if (isYearly) {
        toggle.classList.add("vb-cb-subscription-plans__toggle--active");
        monthlyLabel.classList.remove("vb-cb-subscription-plans__toggle-label--active");
        yearlyLabel.classList.add("vb-cb-subscription-plans__toggle-label--active");
      } else {
        toggle.classList.remove("vb-cb-subscription-plans__toggle--active");
        monthlyLabel.classList.add("vb-cb-subscription-plans__toggle-label--active");
        yearlyLabel.classList.remove("vb-cb-subscription-plans__toggle-label--active");
      }
    }

    if (toggle) {
      toggle.addEventListener("click", function () {
        isYearly = !isYearly;
        updatePrices();
      });
    }
  });
})();
