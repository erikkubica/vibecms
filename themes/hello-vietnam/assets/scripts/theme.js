// Hello Vietnam theme interactions — mirrors the React prototype behaviors.
(function () {
  "use strict";

  // Trips filter (Trips listing page): pills + search filter .trip-card grid items by data-tag / data-title / data-location.
  function initTripsFilter() {
    // Pills/search live in hv-trips-filter block; grid/cards live in hv-trips-grid block — scan globally.
    var root = document.querySelector("[data-trips-root]");
    if (!root && !document.querySelector("[data-trip-grid]")) return;
    var pills = document.querySelectorAll("[data-trip-pill]");
    var searchInput = document.querySelector("[data-trip-search]");
    var grid = document.querySelector("[data-trip-grid]");
    var cards = grid ? grid.querySelectorAll("[data-trip-card]") : [];
    var staffPick = document.querySelector("[data-staff-pick]");
    var countEl = document.querySelector("[data-trip-count]");
    var emptyEl = document.querySelector("[data-trip-empty]");
    var state = { tag: "All", q: "" };

    function apply() {
      var shown = 0;
      cards.forEach(function (c) {
        var tag = c.getAttribute("data-tag") || "";
        var title = (c.getAttribute("data-title") || "").toLowerCase();
        var loc = (c.getAttribute("data-location") || "").toLowerCase();
        var q = state.q.trim().toLowerCase();
        var tagOK = state.tag === "All" || tag === state.tag;
        var qOK = !q || title.indexOf(q) !== -1 || loc.indexOf(q) !== -1;
        if (tagOK && qOK) { c.style.display = ""; shown++; } else { c.style.display = "none"; }
      });
      if (countEl) countEl.textContent = shown + (shown === 1 ? " trip" : " trips");
      if (emptyEl) emptyEl.style.display = shown === 0 ? "" : "none";
      if (staffPick) staffPick.style.display = (state.tag === "All" && !state.q.trim()) ? "" : "none";
    }

    function activatePill(tag) {
      pills.forEach(function (p) {
        var match = (p.getAttribute("data-trip-pill") || "All") === tag;
        p.classList.toggle("active", match);
      });
      state.tag = tag;
    }
    function syncURL() {
      var url = new URL(window.location.href);
      if (state.tag && state.tag !== "All") url.searchParams.set("tag", state.tag);
      else url.searchParams.delete("tag");
      if (state.q.trim()) url.searchParams.set("q", state.q.trim());
      else url.searchParams.delete("q");
      window.history.replaceState(null, "", url.pathname + (url.search ? url.search : "") + url.hash);
    }
    pills.forEach(function (pill) {
      pill.addEventListener("click", function () {
        activatePill(pill.getAttribute("data-trip-pill") || "All");
        syncURL();
        apply();
      });
    });
    if (searchInput) {
      searchInput.addEventListener("input", function () { state.q = searchInput.value; syncURL(); apply(); });
    }
    // Hydrate from URL on load: ?tag=Foodie&q=pho
    var params = new URLSearchParams(window.location.search);
    var urlTag = params.get("tag");
    var urlQ = params.get("q");
    if (urlTag) activatePill(urlTag);
    if (urlQ && searchInput) { searchInput.value = urlQ; state.q = urlQ; }
    apply();
  }

  // Accordion rows (itinerary, FAQs): each [data-accordion] toggles an [data-accordion-panel] sibling.
  function initAccordions() {
    document.querySelectorAll("[data-accordion]").forEach(function (btn) {
      btn.addEventListener("click", function () {
        var wrap = btn.closest("[data-accordion-row]");
        if (!wrap) return;
        var open = wrap.classList.toggle("is-open");
        var panel = wrap.querySelector("[data-accordion-panel]");
        if (panel) panel.style.display = open ? "block" : "none";
        var icon = wrap.querySelector("[data-accordion-icon]");
        if (icon) icon.textContent = open ? "–" : "+";
      });
    });
  }

  // Gallery filter pills + lightbox. Pills and tiles can live in different [data-gallery-root] containers (intro vs masonry).
  function initGallery() {
    var roots = document.querySelectorAll("[data-gallery-root]");
    if (!roots.length) return;
    var pills = document.querySelectorAll("[data-gallery-pill]");
    var tiles = document.querySelectorAll("[data-gallery-tile]");
    pills.forEach(function (pill) {
      pill.addEventListener("click", function () {
        pills.forEach(function (p) { p.classList.remove("active"); });
        pill.classList.add("active");
        var cat = pill.getAttribute("data-gallery-pill") || "All";
        tiles.forEach(function (t) {
          var tc = t.getAttribute("data-category") || "";
          t.style.display = (cat === "All" || tc === cat) ? "" : "none";
        });
      });
    });

    var lightbox = document.querySelector("[data-lightbox]");
    var lightboxImg = lightbox ? lightbox.querySelector("[data-lightbox-img]") : null;
    tiles.forEach(function (tile) {
      tile.style.cursor = "zoom-in";
      tile.addEventListener("click", function () {
        if (!lightbox) return;
        var bg = tile.getAttribute("data-bg") || tile.style.backgroundImage || "";
        var cls = tile.getAttribute("data-ph-variant") || "yellow";
        if (lightboxImg) {
          lightboxImg.className = "ph " + cls + " has-photo";
          lightboxImg.style.backgroundImage = bg;
        }
        lightbox.classList.add("is-open");
      });
    });
    if (lightbox) {
      lightbox.addEventListener("click", function (e) {
        if (e.target === lightbox || e.target.hasAttribute("data-lightbox-close")) {
          lightbox.classList.remove("is-open");
        }
      });
    }
  }

  // Booking form: stepper ± buttons, live total, submit → success state.
  function initBooking() {
    var card = document.querySelector("[data-booking-card]");
    if (!card) return;
    var price = parseFloat(card.getAttribute("data-price") || "0");
    var adultsInput = card.querySelector("[data-adults]");
    var kidsInput = card.querySelector("[data-kids]");
    var totalEls = card.querySelectorAll("[data-total]");

    function recalc() {
      var a = parseInt(adultsInput ? adultsInput.value : "1", 10) || 0;
      var k = parseInt(kidsInput ? kidsInput.value : "0", 10) || 0;
      var total = price * (a + k);
      var label = "$" + total.toFixed(0);
      totalEls.forEach(function (el) { el.textContent = label; });
    }
    card.querySelectorAll("[data-step]").forEach(function (btn) {
      btn.addEventListener("click", function () {
        var target = btn.getAttribute("data-step");
        var delta = parseInt(btn.getAttribute("data-delta") || "0", 10);
        var input = card.querySelector("[data-" + target + "]");
        if (!input) return;
        var v = parseInt(input.value, 10) || 0;
        var min = parseInt(input.getAttribute("min") || "0", 10);
        v = Math.max(min, v + delta);
        input.value = v;
        recalc();
      });
    });
    [adultsInput, kidsInput].forEach(function (i) { if (i) i.addEventListener("input", recalc); });
    recalc();

    // Submission is owned by the forms extension's vibe-form runtime, which
    // performs the actual AJAX POST + validation + success rendering. Theme JS
    // here only handles steppers + live total. The data-booking-success and
    // data-booking-again hooks are kept for theme polish but aren't wired —
    // forms-ext renders its own success state inside .vibe-form-wrapper.
  }

  // Newsletter form (footer): prevent submit, swap to thank-you message.
  function initNewsletter() {
    document.querySelectorAll("[data-newsletter-form]").forEach(function (form) {
      form.addEventListener("submit", function (e) {
        e.preventDefault();
        var input = form.querySelector("input[type=email]");
        if (!input || !input.value) return;
        Array.prototype.forEach.call(form.children, function (c) { c.style.display = "none"; });
        var msg = document.createElement("div");
        msg.style.cssText = "padding: 10px 0; color: var(--green-deep); font-weight: 500;";
        msg.textContent = "Thanks! Check your inbox.";
        msg.setAttribute("data-newsletter-msg", "");
        form.appendChild(msg);
        setTimeout(function () {
          msg.remove();
          Array.prototype.forEach.call(form.children, function (c) { c.style.display = ""; });
          input.value = "";
        }, 4000);
      });
    });
  }

  // Contact form: submit → success.
  function initContactForm() {
    var form = document.querySelector("[data-contact-form]");
    if (!form) return;
    var success = document.querySelector("[data-contact-success]");
    form.addEventListener("submit", function (e) {
      e.preventDefault();
      var n = (form.querySelector("[name=name]") || {}).value;
      var em = (form.querySelector("[name=email]") || {}).value;
      if (!n || !em) return;
      form.style.display = "none";
      if (success) success.style.display = "";
    });
    var again = document.querySelector("[data-contact-again]");
    if (again) {
      again.addEventListener("click", function () {
        form.style.display = "";
        if (success) success.style.display = "none";
        form.reset();
      });
    }
  }

  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", function () {
      initTripsFilter(); initAccordions(); initGallery(); initBooking(); initContactForm(); initNewsletter();
    });
  } else {
    initTripsFilter(); initAccordions(); initGallery(); initBooking(); initContactForm(); initNewsletter();
  }
})();
