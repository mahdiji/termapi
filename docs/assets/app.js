/* TermAPI documentation — small dependency-free enhancement layer. */
(function () {
  "use strict";

  const body = document.body;
  const sidebar = document.getElementById("sidebar");
  const navLinks = Array.from(document.querySelectorAll(".nav-link"));
  const sections = Array.from(document.querySelectorAll(".doc-section"));
  const search = document.getElementById("doc-search");
  const navEmpty = document.querySelector(".nav-empty");
  const progress = document.getElementById("progress");
  const toTop = document.getElementById("to-top");

  /* ---------- mobile navigation ---------- */
  const menuToggle = document.getElementById("menu-toggle");
  if (menuToggle) {
    menuToggle.addEventListener("click", () => {
      body.classList.toggle("nav-open");
      menuToggle.setAttribute("aria-expanded", body.classList.contains("nav-open") ? "true" : "false");
    });
    document.addEventListener("click", (event) => {
      if (!body.classList.contains("nav-open")) return;
      if (sidebar && sidebar.contains(event.target)) return;
      if (menuToggle.contains(event.target)) return;
      body.classList.remove("nav-open");
      menuToggle.setAttribute("aria-expanded", "false");
    });
    navLinks.forEach((link) => {
      link.addEventListener("click", () => {
        body.classList.remove("nav-open");
        menuToggle.setAttribute("aria-expanded", "false");
      });
    });
  }

  /* ---------- search ---------- */
  const sectionByID = new Map();
  sections.forEach((section) => {
    sectionByID.set(section.id, section);
  });

  navLinks.forEach((link) => {
    const id = (link.getAttribute("href") || "").replace(/^#/, "");
    const section = sectionByID.get(id);
    const group = link.closest(".nav-group");
    const groupTitle = group ? (group.querySelector(".nav-title")?.textContent || "") : "";
    const keywords = section ? (section.dataset.keywords || "") : "";
    link.dataset.searchText = [
      link.textContent || "",
      groupTitle,
      keywords,
      section ? (section.dataset.title || "") : "",
    ].join(" ").toLowerCase();
  });

  function applySearch(query) {
    const q = query.trim().toLowerCase();
    let visible = 0;
    navLinks.forEach((link) => {
      const match = !q || link.dataset.searchText.includes(q);
      link.style.display = match ? "" : "none";
      if (match) visible += 1;
    });
    document.querySelectorAll(".nav-group").forEach((group) => {
      const any = Array.from(group.querySelectorAll(".nav-link")).some((link) => link.style.display !== "none");
      group.style.display = any ? "" : "none";
    });
    if (navEmpty) navEmpty.style.display = q && visible === 0 ? "block" : "none";
  }

  if (search) {
    search.addEventListener("input", () => applySearch(search.value));
    search.addEventListener("keydown", (event) => {
      if (event.key === "Escape") {
        search.value = "";
        applySearch("");
        search.blur();
      }
      if (event.key === "Enter") {
        const first = navLinks.find((link) => link.style.display !== "none");
        if (first) {
          const href = first.getAttribute("href");
          if (href) {
            const target = document.querySelector(href);
            if (target) target.scrollIntoView({ behavior: "smooth", block: "start" });
            history.replaceState(null, "", href);
            body.classList.remove("nav-open");
          }
        }
      }
    });
  }

  document.addEventListener("keydown", (event) => {
    const tag = (event.target && event.target.tagName) || "";
    const typing = tag === "INPUT" || tag === "TEXTAREA" || event.target.isContentEditable;
    if (event.key === "/" && !typing && search) {
      event.preventDefault();
      search.focus();
      search.select();
    }
  });

  /* ---------- active section highlighting ---------- */
  if ("IntersectionObserver" in window) {
    const linkByID = new Map();
    navLinks.forEach((link) => {
      const id = (link.getAttribute("href") || "").replace(/^#/, "");
      if (id) linkByID.set(id, link);
    });
    let activeID = "";
    const setActive = (id) => {
      if (id === activeID) return;
      activeID = id;
      navLinks.forEach((link) => link.classList.remove("active"));
      const link = linkByID.get(id);
      if (link) {
        link.classList.add("active");
        const rect = link.getBoundingClientRect();
        const area = sidebar ? sidebar.getBoundingClientRect() : null;
        if (area && (rect.top < area.top + 60 || rect.bottom > area.bottom - 60)) {
          link.scrollIntoView({ block: "nearest" });
        }
      }
    };
    const observer = new IntersectionObserver(
      (entries) => {
        const visible = entries
          .filter((entry) => entry.isIntersecting)
          .sort((a, b) => a.boundingClientRect.top - b.boundingClientRect.top);
        if (visible.length > 0) setActive(visible[0].target.id);
      },
      { rootMargin: "-15% 0px -70% 0px", threshold: [0, 0.1, 0.5] }
    );
    sections.forEach((section) => observer.observe(section));
    if (sections.length > 0) setActive(sections[0].id);
  }

  /* ---------- scroll progress + back to top ---------- */
  let ticking = false;
  function onScroll() {
    if (ticking) return;
    ticking = true;
    requestAnimationFrame(() => {
      const doc = document.documentElement;
      const max = Math.max(doc.scrollHeight - doc.clientHeight, 1);
      const ratio = Math.min(Math.max(doc.scrollTop / max, 0), 1);
      if (progress) progress.style.width = (ratio * 100).toFixed(2) + "%";
      if (toTop) toTop.classList.toggle("show", doc.scrollTop > 600);
      ticking = false;
    });
  }
  window.addEventListener("scroll", onScroll, { passive: true });
  onScroll();
  if (toTop) {
    toTop.addEventListener("click", () => window.scrollTo({ top: 0, behavior: "smooth" }));
  }

  /* ---------- copy code buttons ---------- */
  document.querySelectorAll("pre").forEach((pre) => {
    const code = pre.querySelector("code");
    if (!code) return;
    const wrap = document.createElement("div");
    wrap.className = "code-wrap";
    pre.parentNode.insertBefore(wrap, pre);
    wrap.appendChild(pre);
    const button = document.createElement("button");
    button.type = "button";
    button.className = "copy-btn";
    button.textContent = "copy";
    button.setAttribute("aria-label", "Copy code");
    wrap.appendChild(button);
    button.addEventListener("click", async () => {
      const text = code.innerText;
      try {
        if (navigator.clipboard && window.isSecureContext) {
          await navigator.clipboard.writeText(text);
        } else {
          const area = document.createElement("textarea");
          area.value = text;
          area.style.position = "fixed";
          area.style.opacity = "0";
          document.body.appendChild(area);
          area.select();
          document.execCommand("copy");
          area.remove();
        }
        button.textContent = "copied";
        button.classList.add("done");
        setTimeout(() => {
          button.textContent = "copy";
          button.classList.remove("done");
        }, 1400);
      } catch (_) {
        button.textContent = "failed";
        setTimeout(() => {
          button.textContent = "copy";
        }, 1400);
      }
    });
  });

  /* ---------- theme toggle ---------- */
  const themeToggle = document.getElementById("theme-toggle");
  const themeKey = "termapi-docs-theme";
  function applyTheme(theme) {
    const value = theme === "light" ? "light" : "dark";
    document.documentElement.dataset.theme = value;
    if (themeToggle) {
      themeToggle.textContent = value === "light" ? "Gelap" : "Terang";
      themeToggle.setAttribute("aria-label", value === "light" ? "Aktifkan tema gelap" : "Aktifkan tema terang");
    }
  }
  let stored = null;
  try {
    stored = localStorage.getItem(themeKey);
  } catch (_) {
    stored = null;
  }
  applyTheme(stored || "dark");
  if (themeToggle) {
    themeToggle.addEventListener("click", () => {
      const next = document.documentElement.dataset.theme === "light" ? "dark" : "light";
      applyTheme(next);
      try {
        localStorage.setItem(themeKey, next);
      } catch (_) {
        /* ignore storage failures */
      }
    });
  }

  /* ---------- deep links: apply search reset ---------- */
  window.addEventListener("hashchange", () => {
    if (!search) return;
    if (search.value) {
      search.value = "";
      applySearch("");
    }
  });
})();
