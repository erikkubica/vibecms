/**
 * vibe-form client runtime
 * Handles: conditional field visibility, client-side validation,
 * CAPTCHA token filling, AJAX submission, success/error rendering.
 *
 * Isolated IIFE — safe when multiple forms appear on the same page.
 */
(function () {
  "use strict";

  // ---------------------------------------------------------------------------
  // Condition evaluator — mirrors backend conditions.go exactly
  // ---------------------------------------------------------------------------

  /**
   * Evaluate a condition group against submission data.
   * Returns true when the group is empty (no restriction).
   * Supports nested groups recursively.
   *
   * @param {Object|null} group  - { all?: [...], any?: [...] }
   * @param {Object}      data   - key→value map of current field values
   * @returns {boolean}
   */
  function evaluateGroup(group, data) {
    if (!group || typeof group !== "object") return true;
    var hasAll = Array.isArray(group.all);
    var hasAny = Array.isArray(group.any);
    if (!hasAll && !hasAny) return true;

    if (hasAll) {
      for (var i = 0; i < group.all.length; i++) {
        if (!evaluateItem(group.all[i], data)) return false;
      }
    }
    if (hasAny) {
      var matched = false;
      for (var j = 0; j < group.any.length; j++) {
        if (evaluateItem(group.any[j], data)) { matched = true; break; }
      }
      if (!matched) return false;
    }
    return true;
  }

  function evaluateItem(item, data) {
    if (!item || typeof item !== "object") return false;
    if ("all" in item || "any" in item) return evaluateGroup(item, data);
    return evaluateCondition(item, data);
  }

  /**
   * Evaluate a single condition.
   * Operators: equals, not_equals, contains, not_contains,
   *            gt, gte, lt, lte, in, not_in, matches, is_empty, is_not_empty
   */
  function evaluateCondition(cond, data) {
    var field    = cond.field    || "";
    var op       = cond.operator || "";
    var expected = cond.value;
    var actual   = data.hasOwnProperty(field) ? data[field] : undefined;
    var exists   = actual !== undefined && actual !== null && actual !== "";

    switch (op) {
      case "is_empty":     return !exists || condIsEmpty(actual);
      case "is_not_empty": return exists && !condIsEmpty(actual);
      case "equals":       return String(actual) === String(expected);
      case "not_equals":   return String(actual) !== String(expected);
      case "contains":
        return String(actual).toLowerCase().indexOf(String(expected).toLowerCase()) !== -1;
      case "not_contains":
        return String(actual).toLowerCase().indexOf(String(expected).toLowerCase()) === -1;
      case "gt":  return toNum(actual) >  toNum(expected);
      case "gte": return toNum(actual) >= toNum(expected);
      case "lt":  return toNum(actual) <  toNum(expected);
      case "lte": return toNum(actual) <= toNum(expected);
      case "in":
        if (!Array.isArray(expected)) return false;
        for (var i = 0; i < expected.length; i++) {
          if (String(expected[i]) === String(actual)) return true;
        }
        return false;
      case "not_in":
        if (!Array.isArray(expected)) return true;
        for (var j = 0; j < expected.length; j++) {
          if (String(expected[j]) === String(actual)) return false;
        }
        return true;
      case "matches":
        try {
          return new RegExp(String(expected)).test(String(actual));
        } catch (e) {
          console.warn("[vibe-form] Invalid regex in condition:", expected);
          return false;
        }
      default:
        return false;
    }
  }

  function condIsEmpty(v) {
    if (v === null || v === undefined) return true;
    if (typeof v === "string") return v.trim() === "";
    if (typeof v === "boolean") return !v;
    if (Array.isArray(v)) return v.length === 0;
    return false;
  }

  function toNum(v) {
    var n = parseFloat(String(v));
    return isNaN(n) ? 0 : n;
  }

  // ---------------------------------------------------------------------------
  // Per-form initialisation
  // ---------------------------------------------------------------------------

  function collectFormData(form) {
    var data = {};
    new FormData(form).forEach(function (value, key) {
      if (data.hasOwnProperty(key)) {
        if (!Array.isArray(data[key])) data[key] = [data[key]];
        data[key].push(value);
      } else {
        data[key] = value;
      }
    });
    return data;
  }

  function applyVisibility(form, fields) {
    var currentData = collectFormData(form);
    fields.forEach(function (f) {
      if (!f.id || !f.display_when) return;
      var visible = evaluateGroup(f.display_when, currentData);
      // Find the field container — walk up from the input to the nearest .vibe-field wrapper
      var input = form.querySelector("[name=\"" + f.id + "\"]");
      if (!input) return;
      var container = input.closest(".vibe-field, .vf-field, [data-field-id=\"" + f.id + "\"]") || input.parentElement;
      if (!container) return;
      container.style.display = visible ? "" : "none";
    });
  }

  function vibeFormValidate(form, fields) {
    var errors = {};
    var currentData = collectFormData(form);

    fields.forEach(function (f) {
      if (!f.id) return;

      // Skip hidden (conditional) fields
      if (f.display_when && !evaluateGroup(f.display_when, currentData)) return;

      var input = form.querySelector("[name=\"" + f.id + "\"]");
      if (!input) return;
      var val = input.type === "checkbox" ? input.checked : input.value;

      if (f.required) {
        if (input.type === "file") {
          if (!input.files || input.files.length === 0) {
            errors[f.id] = (f.label || f.id) + " is required";
            return;
          }
        } else if (input.type === "checkbox") {
          if (!val) {
            errors[f.id] = "You must agree to the privacy policy";
            return;
          }
        } else if (val === "" || val === null || val === undefined) {
          errors[f.id] = (f.label || f.id) + " is required";
          return;
        }
      }

      if (input.type === "file") return;

      var strVal = typeof val === "string" ? val : String(val);

      if (f.min_length && strVal.length < f.min_length) {
        errors[f.id] = "Minimum " + f.min_length + " characters";
        return;
      }
      if (f.max_length && strVal.length > f.max_length) {
        errors[f.id] = "Maximum " + f.max_length + " characters";
        return;
      }

      if (f.type === "number" || f.type === "range") {
        var num = parseFloat(strVal);
        if (!isNaN(num)) {
          if (f.min !== undefined && f.min !== null && num < f.min) {
            errors[f.id] = "Must be at least " + f.min;
            return;
          }
          if (f.max !== undefined && f.max !== null && num > f.max) {
            errors[f.id] = "Must be at most " + f.max;
            return;
          }
          if (f.step && f.step > 0) {
            var rem = Math.abs(num % f.step);
            if (rem > 1e-9 && Math.abs(rem - f.step) > 1e-9) {
              errors[f.id] = "Must be a multiple of " + f.step;
              return;
            }
          }
        }
      }

      if (f.pattern && strVal !== "") {
        try {
          if (!new RegExp(f.pattern).test(strVal)) {
            errors[f.id] = "Invalid format";
            return;
          }
        } catch (e) { /* invalid pattern — skip */ }
      }

      if (f.type === "email" && strVal !== "") {
        if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(strVal)) {
          errors[f.id] = "Invalid email format";
          return;
        }
      }
    });
    return errors;
  }

  function vibeFormRenderErrors(form, errors) {
    form.querySelectorAll(".field-error").forEach(function (e) { e.remove(); });
    form.querySelectorAll("[aria-invalid]").forEach(function (e) { e.removeAttribute("aria-invalid"); });
    Object.keys(errors).forEach(function (id) {
      var msg   = errors[id];
      var input = form.querySelector("[name=\"" + id + "\"]");
      if (!input) return;
      input.setAttribute("aria-invalid", "true");
      var err = document.createElement("div");
      err.className = "field-error";
      err.textContent = msg;
      input.parentNode.insertBefore(err, input.nextSibling);
    });
    var first = form.querySelector("[aria-invalid]");
    if (first) first.scrollIntoView({ behavior: "smooth", block: "center" });
  }

  async function vibeFormFillCaptchaToken(form) {
    var input = form.querySelector("input[name=\"_captcha_token\"]");
    if (!input) return;
    var provider = input.dataset.captcha;
    if (provider === "recaptcha") {
      input.value = await new Promise(function (resolve) {
        grecaptcha.ready(function () {
          grecaptcha.execute(input.dataset.sitekey, { action: "submit" }).then(resolve);
        });
      });
    } else if (provider === "hcaptcha") {
      input.value = (window.hcaptcha && window.hcaptcha.getResponse()) || "";
    }
    // turnstile: callback already wrote input.value
  }

  function initForm(wrapper) {
    var form = wrapper.querySelector("form");
    if (!form) return;

    var formFields = [];
    var formSettings = {};
    var formSlug = wrapper.dataset.formSlug || "";

    var metaEl = wrapper.querySelector("[data-vibe-form-meta]");
    if (metaEl) {
      try {
        var meta = JSON.parse(metaEl.textContent || metaEl.innerText || "{}");
        formFields   = meta.fields   || [];
        formSettings = meta.settings || {};
      } catch (e) { /* skip if meta missing */ }
    }

    // Initial visibility pass
    if (formFields.length > 0) {
      applyVisibility(form, formFields);

      // Re-evaluate on every input/change event
      form.addEventListener("input",  function () { applyVisibility(form, formFields); });
      form.addEventListener("change", function () { applyVisibility(form, formFields); });
    }

    form.addEventListener("submit", async function (e) {
      e.preventDefault();

      if (formFields.length > 0) {
        var clientErrors = vibeFormValidate(form, formFields);
        if (Object.keys(clientErrors).length > 0) {
          vibeFormRenderErrors(form, clientErrors);
          return;
        }
      }

      form.querySelectorAll(".field-error").forEach(function (el) { el.remove(); });
      form.querySelectorAll("[aria-invalid]").forEach(function (el) { el.removeAttribute("aria-invalid"); });

      var submitBtn = form.querySelector("[type=\"submit\"]");
      var originalBtnText = submitBtn ? submitBtn.innerHTML : "Submit";
      if (submitBtn) {
        submitBtn.disabled = true;
        submitBtn.innerHTML = "<span class=\"spinner\"></span> Submitting...";
      }

      try {
        await vibeFormFillCaptchaToken(form);

        var hasFiles = !!form.querySelector("input[type=\"file\"]");
        var body, headers;
        if (hasFiles) {
          body    = new FormData(form);
          headers = {};
        } else {
          var data = collectFormData(form);
          body    = JSON.stringify(data);
          headers = { "Content-Type": "application/json" };
        }

        var response = await fetch("/forms/submit/" + formSlug, {
          method:      "POST",
          headers:     headers,
          body:        body,
          credentials: "same-origin",
        });

        var result = await response.json();

        if (response.ok) {
          var successMsg = (formSettings && formSettings.success_message)
            ? formSettings.success_message
            : (result.message || "Thank you! Your submission has been received.");
          if (formSettings && formSettings.redirect_url) {
            window.location = formSettings.redirect_url;
          } else {
            form.innerHTML = "<div class=\"form-success-message\">" + successMsg + "</div>";
          }
        } else if (result.fields) {
          vibeFormRenderErrors(form, result.fields);
        } else {
          var errMsg = (formSettings && formSettings.error_message)
            ? formSettings.error_message
            : (result.message || result.error || "Something went wrong. Please try again.");
          alert(errMsg);
        }
      } catch (error) {
        console.error("[vibe-form] Submission error:", error);
        alert("Failed to submit form. Please check your connection.");
      } finally {
        if (submitBtn) {
          submitBtn.disabled  = false;
          submitBtn.innerHTML = originalBtnText;
        }
      }
    });
  }

  // Turnstile global callback
  window.vibeFormTurnstileCallback = function (token) {
    document.querySelectorAll("input[name=\"_captcha_token\"][data-captcha=\"turnstile\"]").forEach(function (i) {
      i.value = token;
    });
  };

  // Initialise all forms when DOM is ready
  function boot() {
    document.querySelectorAll(".vibe-form-wrapper").forEach(initForm);
  }

  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", boot);
  } else {
    boot();
  }
})();
