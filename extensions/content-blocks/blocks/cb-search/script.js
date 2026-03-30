(function () {
  document.querySelectorAll('[data-block="cb-search"]').forEach(function (block) {
    var nodeType = block.getAttribute("data-node-type") || "page";
    var limit = parseInt(block.getAttribute("data-limit"), 10) || 5;
    var input = block.querySelector(".vb-cb-search__input");
    var results = block.querySelector(".vb-cb-search__results");
    var spinner = block.querySelector(".vb-cb-search__spinner");
    var debounceTimer = null;

    if (!input || !results || !spinner) return;

    function renderResults(nodes, query) {
      while (results.firstChild) results.removeChild(results.firstChild);

      if (nodes.length === 0) {
        var empty = document.createElement("div");
        empty.className = "vb-cb-search__no-results";
        empty.textContent = "No results found for \u201C" + query + "\u201D";
        results.appendChild(empty);
        results.style.display = "block";
        return;
      }

      nodes.forEach(function (node) {
        var a = document.createElement("a");
        a.href = "/" + (node.slug || "");
        a.className = "vb-cb-search__result-item";

        var title = document.createElement("div");
        title.className = "vb-cb-search__result-title";
        title.textContent = node.title || "Untitled";
        a.appendChild(title);

        var excerpt = node.excerpt || "";
        if (excerpt.length > 120) excerpt = excerpt.substring(0, 120) + "...";
        var desc = document.createElement("div");
        desc.className = "vb-cb-search__result-excerpt";
        desc.textContent = excerpt;
        a.appendChild(desc);

        results.appendChild(a);
      });

      results.style.display = "block";
    }

    function showMessage(msg) {
      while (results.firstChild) results.removeChild(results.firstChild);
      var el = document.createElement("div");
      el.className = "vb-cb-search__no-results";
      el.textContent = msg;
      results.appendChild(el);
      results.style.display = "block";
    }

    input.addEventListener("input", function () {
      clearTimeout(debounceTimer);
      var query = input.value.trim();

      if (query.length < 2) {
        results.style.display = "none";
        while (results.firstChild) results.removeChild(results.firstChild);
        spinner.style.display = "none";
        return;
      }

      spinner.style.display = "block";

      debounceTimer = setTimeout(function () {
        fetch(
          "/admin/api/nodes?node_type=" +
            encodeURIComponent(nodeType) +
            "&search=" +
            encodeURIComponent(query) +
            "&limit=" +
            limit
        )
          .then(function (res) {
            if (res.status === 401 || res.status === 403) {
              spinner.style.display = "none";
              showMessage("Sign in to preview dynamic content.");
              return null;
            }
            if (!res.ok) {
              spinner.style.display = "none";
              showMessage("Search unavailable. Please try again.");
              return null;
            }
            return res.json();
          })
          .then(function (json) {
            if (!json) return;
            spinner.style.display = "none";
            renderResults(json.data || [], query);
          })
          .catch(function () {
            spinner.style.display = "none";
            showMessage("Search unavailable. Please try again.");
          });
      }, 300);
    });

    document.addEventListener("click", function (e) {
      if (!block.contains(e.target)) {
        results.style.display = "none";
      }
    });

    input.addEventListener("focus", function () {
      if (results.childNodes.length > 0) {
        results.style.display = "block";
      }
    });
  });
})();
