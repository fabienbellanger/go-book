// theme.js — bascule clair/sombre, persistée dans localStorage.
// Le thème initial est posé très tôt par un script inline dans <head> (anti-FOUC) ;
// ce fichier ne gère que le bouton de bascule.
(() => {
  const current = () =>
    document.documentElement.getAttribute("data-theme") || "light";

  const apply = (theme) => {
    document.documentElement.setAttribute("data-theme", theme);
    try {
      localStorage.setItem("theme", theme);
    } catch {}
  };

  const initThemeToggle = () => {
    const btn = document.getElementById("theme-toggle");
    if (!btn) return;
    btn.addEventListener("click", () => {
      apply(current() === "dark" ? "light" : "dark");
    });
  };

  // Mémorise la dernière page lue (lue depuis data-last-page sur <body>) pour y
  // revenir à la réouverture du site ; la redirection est faite par index.html.
  const rememberLastPage = () => {
    const last = document.body.dataset.lastPage;
    if (!last) return;
    try {
      localStorage.setItem("lastPage", last);
    } catch {}
  };

  // Masquage du sommaire de gauche. Sur mobile (≤800px) le bouton ouvre/ferme le
  // tiroir ; sur desktop il replie la colonne (état persisté dans localStorage).
  const initSidebarToggle = () => {
    const btn = document.getElementById("sidebar-toggle");
    if (!btn) return;

    // Restaure l'état desktop sauvegardé.
    try {
      if (localStorage.getItem("sidebar") === "collapsed") {
        document.body.classList.add("sidebar-collapsed");
      }
    } catch {}

    btn.addEventListener("click", () => {
      if (window.matchMedia("(max-width: 800px)").matches) {
        document.body.classList.toggle("nav-open");
        return;
      }
      const collapsed = document.body.classList.toggle("sidebar-collapsed");
      try {
        localStorage.setItem("sidebar", collapsed ? "collapsed" : "shown");
      } catch {}
    });
  };

  // Bouton « Tout déplier / Tout replier » : ouvre ou ferme toutes les parties
  // du sommaire d'un coup. Le libellé reflète l'action à venir.
  const initExpandAll = () => {
    const btn = document.getElementById("toggle-all-parts");
    if (!btn) return;

    const parts = () => document.querySelectorAll(".sidebar .nav-part");
    const refresh = () => {
      const ps = parts();
      const allOpen =
        ps.length > 0 && Array.prototype.every.call(ps, (p) => p.open);
      btn.textContent = allOpen ? "Tout replier" : "Tout déplier";
      btn.dataset.mode = allOpen ? "collapse" : "expand";
    };

    btn.addEventListener("click", () => {
      const expand = btn.dataset.mode !== "collapse";
      Array.prototype.forEach.call(parts(), (p) => {
        p.open = expand;
      });
      refresh();
    });
    // Garde le libellé synchronisé si l'utilisateur ouvre/ferme une partie.
    Array.prototype.forEach.call(parts(), (p) => {
      p.addEventListener("toggle", refresh);
    });
    refresh();
  };

  const init = () => {
    initThemeToggle();
    rememberLastPage();
    initSidebarToggle();
    initExpandAll();
  };

  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", init);
  } else {
    init();
  }
})();
