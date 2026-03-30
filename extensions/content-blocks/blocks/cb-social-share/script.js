(function () {
  document.querySelectorAll('[data-block="cb-social-share"]').forEach(function (block) {
    var buttons = block.querySelectorAll(".vb-cb-social-share__btn");
    var pageUrl = encodeURIComponent(window.location.href);
    var pageTitle = encodeURIComponent(document.title);

    var shareUrls = {
      twitter:
        "https://twitter.com/intent/tweet?url=" + pageUrl + "&text=" + pageTitle,
      facebook:
        "https://www.facebook.com/sharer/sharer.php?u=" + pageUrl,
      linkedin:
        "https://www.linkedin.com/sharing/share-offsite/?url=" + pageUrl,
      email:
        "mailto:?subject=" + pageTitle + "&body=" + pageUrl,
    };

    buttons.forEach(function (btn) {
      btn.addEventListener("click", function () {
        var platform = btn.getAttribute("data-share");

        if (platform === "copy") {
          navigator.clipboard
            .writeText(window.location.href)
            .then(function () {
              var copyIcon = btn.querySelector(".vb-cb-social-share__copy-icon");
              var checkIcon = btn.querySelector(".vb-cb-social-share__check-icon");
              var label = btn.querySelector(".vb-cb-social-share__btn-label");

              btn.classList.add("vb-cb-social-share__btn--copied");
              if (copyIcon) copyIcon.style.display = "none";
              if (checkIcon) checkIcon.style.display = "inline";
              if (label) label.textContent = "Copied!";

              setTimeout(function () {
                btn.classList.remove("vb-cb-social-share__btn--copied");
                if (copyIcon) copyIcon.style.display = "inline";
                if (checkIcon) checkIcon.style.display = "none";
                if (label) label.textContent = "Copy Link";
              }, 2000);
            });
          return;
        }

        if (platform === "email") {
          window.location.href = shareUrls[platform];
          return;
        }

        if (shareUrls[platform]) {
          window.open(shareUrls[platform], "_blank", "width=600,height=400,scrollbars=yes");
        }
      });
    });
  });
})();
