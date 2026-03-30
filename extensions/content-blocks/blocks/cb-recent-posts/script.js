(function () {
  document.querySelectorAll('[data-block="cb-recent-posts"]').forEach(function (block) {
    var nodeType = block.getAttribute("data-node-type") || "post";
    var count = parseInt(block.getAttribute("data-count"), 10) || 6;
    var showImage = block.getAttribute("data-show-image") === "true";
    var showExcerpt = block.getAttribute("data-show-excerpt") === "true";
    var showDate = block.getAttribute("data-show-date") === "true";
    var linkText = block.getAttribute("data-link-text") || "Read More";
    var container = block.querySelector(".vb-cb-recent-posts__container");

    if (!container) return;

    function formatDate(dateStr) {
      if (!dateStr) return "";
      var d = new Date(dateStr);
      return d.toLocaleDateString("en-US", {
        year: "numeric",
        month: "short",
        day: "numeric",
      });
    }

    function buildCard(node) {
      var card = document.createElement("article");
      card.className = "vb-cb-recent-posts__card";

      if (showImage && node.featured_image) {
        var img = document.createElement("img");
        img.className = "vb-cb-recent-posts__image";
        img.src = node.featured_image;
        img.alt = node.title || "";
        img.loading = "lazy";
        card.appendChild(img);
      }

      var body = document.createElement("div");
      body.className = "vb-cb-recent-posts__body";

      if (showDate && node.published_at) {
        var date = document.createElement("time");
        date.className = "vb-cb-recent-posts__date";
        date.textContent = formatDate(node.published_at);
        body.appendChild(date);
      }

      var title = document.createElement("h3");
      title.className = "vb-cb-recent-posts__title";
      title.textContent = node.title || "Untitled";
      body.appendChild(title);

      if (showExcerpt && node.excerpt) {
        var excerpt = document.createElement("p");
        excerpt.className = "vb-cb-recent-posts__excerpt";
        var text = node.excerpt;
        if (text.length > 150) text = text.substring(0, 150) + "...";
        excerpt.textContent = text;
        body.appendChild(excerpt);
      }

      var link = document.createElement("a");
      link.className = "vb-cb-recent-posts__link";
      link.href = "/" + (node.slug || "");
      link.textContent = linkText;
      body.appendChild(link);

      card.appendChild(body);
      return card;
    }

    function showMessage(msg) {
      while (container.firstChild) container.removeChild(container.firstChild);
      var p = document.createElement("p");
      p.style.textAlign = "center";
      p.style.color = "var(--color-text-muted)";
      p.textContent = msg;
      container.appendChild(p);
    }

    fetch(
      "/admin/api/nodes?node_type=" +
        encodeURIComponent(nodeType) +
        "&limit=" +
        count +
        "&status=published&sort=-published_at"
    )
      .then(function (res) {
        if (res.status === 401 || res.status === 403) {
          showMessage("Sign in to preview dynamic content.");
          return null;
        }
        if (!res.ok) {
          showMessage("Unable to load posts. Please refresh the page.");
          return null;
        }
        return res.json();
      })
      .then(function (json) {
        if (!json) return;
        var nodes = json.data || [];
        while (container.firstChild) container.removeChild(container.firstChild);

        if (nodes.length === 0) {
          showMessage("No posts found.");
          return;
        }

        nodes.forEach(function (node) {
          container.appendChild(buildCard(node));
        });
      })
      .catch(function () {
        showMessage("Unable to load posts. Please refresh the page.");
      });
  });
})();
