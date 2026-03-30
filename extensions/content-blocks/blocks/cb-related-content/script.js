(function () {
  document.querySelectorAll('[data-block="cb-related-content"]').forEach(function (block) {
    var nodeType = block.getAttribute("data-node-type") || "post";
    var count = parseInt(block.getAttribute("data-count"), 10) || 4;
    var showImage = block.getAttribute("data-show-image") === "true";
    var track = block.querySelector(".vb-cb-related-content__track");

    /* Drag-to-scroll for mouse users */
    var isDragging = false;
    var startX = 0;
    var scrollLeft = 0;

    track.addEventListener("mousedown", function (e) {
      isDragging = true;
      startX = e.pageX - track.offsetLeft;
      scrollLeft = track.scrollLeft;
      track.style.cursor = "grabbing";
    });

    track.addEventListener("mouseleave", function () {
      isDragging = false;
      track.style.cursor = "grab";
    });

    track.addEventListener("mouseup", function () {
      isDragging = false;
      track.style.cursor = "grab";
    });

    track.addEventListener("mousemove", function (e) {
      if (!isDragging) return;
      e.preventDefault();
      var x = e.pageX - track.offsetLeft;
      var walk = (x - startX) * 1.5;
      track.scrollLeft = scrollLeft - walk;
    });

    track.style.cursor = "grab";

    function formatDate(dateStr) {
      if (!dateStr) return "";
      var d = new Date(dateStr);
      return d.toLocaleDateString("en-US", { month: "short", day: "numeric" });
    }

    function buildCard(node) {
      var a = document.createElement("a");
      a.href = "/" + (node.slug || "");
      a.className = "vb-cb-related-content__card";

      if (showImage && node.featured_image) {
        var img = document.createElement("img");
        img.className = "vb-cb-related-content__card-image";
        img.src = node.featured_image;
        img.alt = node.title || "";
        img.loading = "lazy";
        a.appendChild(img);
      }

      var body = document.createElement("div");
      body.className = "vb-cb-related-content__card-body";

      var title = document.createElement("span");
      title.className = "vb-cb-related-content__card-title";
      title.textContent = node.title || "Untitled";
      body.appendChild(title);

      if (node.published_at) {
        var meta = document.createElement("span");
        meta.className = "vb-cb-related-content__card-meta";
        meta.textContent = formatDate(node.published_at);
        body.appendChild(meta);
      }

      a.appendChild(body);
      return a;
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
        while (track.firstChild) track.removeChild(track.firstChild);

        if (nodes.length === 0) {
          var empty = document.createElement("p");
          empty.style.color = "var(--color-text-muted)";
          empty.textContent = "No related content found.";
          track.appendChild(empty);
          return;
        }

        nodes.forEach(function (node) {
          track.appendChild(buildCard(node));
        });
      })
      .catch(function () {
        while (track.firstChild) track.removeChild(track.firstChild);
        var err = document.createElement("p");
        err.style.color = "var(--color-text-muted)";
        err.textContent = "Unable to load related content.";
        track.appendChild(err);
      });
  });
})();
