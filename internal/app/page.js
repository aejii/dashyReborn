(() => {
  const qs = (s, root = document) => Array.from(root.querySelectorAll(s));

  const search = document.getElementById("dashboard-search");
  const empty = document.getElementById("search-empty");
  if (search) {
    const applyFilter = () => {
      const query = search.value.trim().toLowerCase();
      qs("[data-filter]").forEach((node) => {
        const haystack = (node.getAttribute("data-filter") || "").toLowerCase();
        node.hidden = query !== "" && !haystack.includes(query);
      });
      let anyVisible = false;
      qs("[data-section]").forEach((section) => {
        const visibleChildren = qs("[data-filter]", section).some((node) => !node.hidden);
        section.hidden = !visibleChildren;
        if (visibleChildren) anyVisible = true;
      });
      if (empty) empty.hidden = anyVisible;
    };
    search.addEventListener("input", applyFilter);
    applyFilter();
  }

  qs(".toggle[data-key]").forEach((toggle) => {
    const key = "dashyreborn:collapse:" + toggle.dataset.key;
    const saved = window.localStorage.getItem(key);
    if (saved !== null) toggle.checked = saved === "1";
    toggle.addEventListener("change", () => {
      window.localStorage.setItem(key, toggle.checked ? "1" : "0");
    });
  });

  if (window.EventSource) {
    const stream = new EventSource("/events");
    stream.addEventListener("reload", () => window.location.reload());
  }
})();
