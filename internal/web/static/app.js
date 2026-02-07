// Prevent native form submission for htmx-only forms.
document.addEventListener("submit", function (e) {
  if (e.target.hasAttribute("data-no-submit")) {
    e.preventDefault();
  }
});

// Handle "go back" navigation via data attribute.
document.addEventListener("click", function (e) {
  if (e.target.closest("[data-go-back]")) {
    e.preventDefault();
    history.back();
  }
});
