(function () {
  document.querySelectorAll('[data-block="cb-recent-posts"]').forEach(function (block) {
    var nodeType = block.getAttribute("data-node-type") || "post";
    var count = parseInt(block.getAttribute("data-count"), 10) || 6;
    var showImage = block.getAttribute("data-show-image") === "true";
    var showExcerpt = block.getAttribute("data-show-excerpt") === "true";
    var showDate = block.getAttribute("data-show-date") === "true";
    var linkText = block.getAttribute("data-link-text") || "Read More";
    var container = block.querySelector(".vb-cb-recent-posts__container");

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

    fetch(
      "/api/v1/nodes?type=" +
        encodeURIComponent(nodeType) +
        "&limit=" +
        count +
        "&status=published&sort=-published_at"
    )
      .then(function (res) {
        return res.json();
      })
      .then(function (json) {
        var nodes = json.data || [];
        while (container.firstChild) container.removeChild(container.firstChild);

        if (nodes.length === 0) {
          var empty = document.createElement("p");
          empty.style.textAlign = "center";
          empty.style.color = "var(--color-text-muted)";
          empty.textContent = "No posts found.";
          container.appendChild(empty);
          return;
        }

        nodes.forEach(function (node) {
          container.appendChild(buildCard(node));
        });
      })
      .catch(function () {
        while (container.firstChild) container.removeChild(container.firstChild);
        var err = document.createElement("p");
        err.style.textAlign = "center";
        err.style.color = "var(--color-text-muted)";
        err.textContent = "Unable to load posts. Please refresh the page.";
        container.appendChild(err);
      });
  });
})();
