// theme.js — bascule clair/sombre, persistée dans localStorage.
// Le thème initial est posé très tôt par un script inline dans <head> (anti-FOUC) ;
// ce fichier ne gère que le bouton de bascule.
(function () {
  "use strict";

  function current() {
    return document.documentElement.getAttribute("data-theme") || "light";
  }

  function apply(theme) {
    document.documentElement.setAttribute("data-theme", theme);
    try {
      localStorage.setItem("theme", theme);
    } catch (e) {}
  }

  function initThemeToggle() {
    var btn = document.getElementById("theme-toggle");
    if (!btn) return;
    btn.addEventListener("click", function () {
      apply(current() === "dark" ? "light" : "dark");
    });
  }

  // Masquage du sommaire de gauche. Sur mobile (≤800px) le bouton ouvre/ferme le
  // tiroir ; sur desktop il replie la colonne (état persisté dans localStorage).
  function initSidebarToggle() {
    var btn = document.getElementById("sidebar-toggle");
    if (!btn) return;

    // Restaure l'état desktop sauvegardé.
    try {
      if (localStorage.getItem("sidebar") === "collapsed") {
        document.body.classList.add("sidebar-collapsed");
      }
    } catch (e) {}

    btn.addEventListener("click", function () {
      if (window.matchMedia("(max-width: 800px)").matches) {
        document.body.classList.toggle("nav-open");
        return;
      }
      var collapsed = document.body.classList.toggle("sidebar-collapsed");
      try {
        localStorage.setItem("sidebar", collapsed ? "collapsed" : "shown");
      } catch (e) {}
    });
  }

  // Bouton « Tout déplier / Tout replier » : ouvre ou ferme toutes les parties
  // du sommaire d'un coup. Le libellé reflète l'action à venir.
  function initExpandAll() {
    var btn = document.getElementById("toggle-all-parts");
    if (!btn) return;

    function parts() {
      return document.querySelectorAll(".sidebar .nav-part");
    }
    function refresh() {
      var ps = parts();
      var allOpen = ps.length > 0 &&
        Array.prototype.every.call(ps, function (p) { return p.open; });
      btn.textContent = allOpen ? "Tout replier" : "Tout déplier";
      btn.dataset.mode = allOpen ? "collapse" : "expand";
    }

    btn.addEventListener("click", function () {
      var expand = btn.dataset.mode !== "collapse";
      Array.prototype.forEach.call(parts(), function (p) { p.open = expand; });
      refresh();
    });
    // Garde le libellé synchronisé si l'utilisateur ouvre/ferme une partie.
    Array.prototype.forEach.call(parts(), function (p) {
      p.addEventListener("toggle", refresh);
    });
    refresh();
  }

  function init() {
    initThemeToggle();
    initSidebarToggle();
    initExpandAll();
  }

  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", init);
  } else {
    init();
  }
})();
