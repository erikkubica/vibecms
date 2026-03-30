(function () {
  document.querySelectorAll('[data-block="cb-contact-form"]').forEach(function (block) {
    var form = block.querySelector(".vb-cb-contact-form__form");
    var actionUrl = block.getAttribute("data-action");
    var successMsg = block.getAttribute("data-success") || "Thank you!";
    var submitBtn = form.querySelector(".vb-cb-contact-form__button");
    var btnText = form.querySelector(".vb-cb-contact-form__button-text");
    var btnSpinner = form.querySelector(".vb-cb-contact-form__button-spinner");
    var feedback = form.querySelector(".vb-cb-contact-form__feedback");

    function validateEmail(email) {
      return /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email);
    }

    function showError(fieldName, message) {
      var input = form.querySelector('[name="' + fieldName + '"]');
      var errorEl = form.querySelector('[data-field="' + fieldName + '"]');
      if (input) input.classList.add("vb-cb-contact-form__input--error");
      if (errorEl) errorEl.textContent = message;
    }

    function clearErrors() {
      form.querySelectorAll(".vb-cb-contact-form__input--error").forEach(function (el) {
        el.classList.remove("vb-cb-contact-form__input--error");
      });
      form.querySelectorAll(".vb-cb-contact-form__error").forEach(function (el) {
        el.textContent = "";
      });
    }

    function validate() {
      clearErrors();
      var valid = true;
      var name = form.querySelector('[name="name"]');
      var email = form.querySelector('[name="email"]');
      var message = form.querySelector('[name="message"]');

      if (name && !name.value.trim()) {
        showError("name", "Name is required.");
        valid = false;
      }
      if (email && !email.value.trim()) {
        showError("email", "Email is required.");
        valid = false;
      } else if (email && !validateEmail(email.value.trim())) {
        showError("email", "Please enter a valid email.");
        valid = false;
      }
      if (message && !message.value.trim()) {
        showError("message", "Message is required.");
        valid = false;
      }
      return valid;
    }

    function setLoading(loading) {
      submitBtn.disabled = loading;
      btnText.style.display = loading ? "none" : "inline";
      btnSpinner.style.display = loading ? "inline" : "none";
    }

    function showFeedback(type, message) {
      feedback.textContent = message;
      feedback.className =
        "vb-cb-contact-form__feedback vb-cb-contact-form__feedback--" + type;
      feedback.style.display = "block";
    }

    form.addEventListener("submit", function (e) {
      e.preventDefault();
      feedback.style.display = "none";

      if (!validate()) return;

      setLoading(true);

      var data = {};
      var formData = new FormData(form);
      formData.forEach(function (value, key) {
        data[key] = value;
      });

      fetch(actionUrl, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(data),
      })
        .then(function (res) {
          setLoading(false);
          if (res.ok) {
            showFeedback("success", successMsg);
            form.reset();
            clearErrors();
          } else {
            showFeedback("error", "Something went wrong. Please try again.");
          }
        })
        .catch(function () {
          setLoading(false);
          showFeedback("error", "Network error. Please check your connection.");
        });
    });

    /* Clear error on input */
    form.querySelectorAll(".vb-cb-contact-form__input").forEach(function (input) {
      input.addEventListener("input", function () {
        input.classList.remove("vb-cb-contact-form__input--error");
        var errorEl = form.querySelector('[data-field="' + input.name + '"]');
        if (errorEl) errorEl.textContent = "";
      });
    });
  });
})();
