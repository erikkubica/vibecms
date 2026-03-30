(function () {
  function parseVideoUrl(url) {
    if (!url) return null;

    // YouTube
    var ytMatch = url.match(
      /(?:youtube\.com\/(?:watch\?v=|embed\/)|youtu\.be\/)([a-zA-Z0-9_-]{11})/
    );
    if (ytMatch) {
      return "https://www.youtube.com/embed/" + ytMatch[1] + "?autoplay=1&rel=0";
    }

    // Vimeo
    var vimeoMatch = url.match(/vimeo\.com\/(\d+)/);
    if (vimeoMatch) {
      return "https://player.vimeo.com/video/" + vimeoMatch[1] + "?autoplay=1";
    }

    return url;
  }

  function initVideoEmbed(block) {
    var container = block.querySelector(".vb-cb-video-embed__container");
    if (!container) return;

    var videoUrl = container.getAttribute("data-video-url");
    var poster = container.querySelector(".vb-cb-video-embed__poster");

    // If no poster, load iframe immediately
    if (!poster) {
      var iframe = container.querySelector(".vb-cb-video-embed__iframe");
      if (iframe) {
        var embedUrl = parseVideoUrl(videoUrl);
        if (embedUrl) {
          // Remove autoplay for direct embeds (no poster click)
          iframe.src = embedUrl.replace("autoplay=1", "autoplay=0");
        }
      }
      return;
    }

    // Click poster to load video
    poster.addEventListener("click", function () {
      var embedUrl = parseVideoUrl(videoUrl);
      if (!embedUrl) return;

      var iframe = document.createElement("iframe");
      iframe.className = "vb-cb-video-embed__iframe";
      iframe.src = embedUrl;
      iframe.setAttribute("frameborder", "0");
      iframe.setAttribute("allow", "accelerometer; autoplay; clipboard-write; encrypted-media; gyroscope; picture-in-picture");
      iframe.setAttribute("allowfullscreen", "");

      container.replaceChild(iframe, poster);
    });
  }

  document.querySelectorAll('[data-block="cb-video-embed"]').forEach(initVideoEmbed);
})();
