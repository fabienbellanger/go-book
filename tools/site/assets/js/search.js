// search.js — recherche plein-texte côté client, sans dépendance.
//
// Charge search-index.json au premier usage, filtre par sous-chaîne normalisée
// (insensible aux accents et à la casse), classe les résultats (titre > section
// > corps) et les affiche : en surimpression depuis le header, et en liste pleine
// page sur search.html. Navigation clavier ↑/↓/Entrée, focus rapide avec « / ».
(function () {
  "use strict";

  var base = "";
  var script = document.querySelector('script[src*="search.js"]');
  if (script) base = script.getAttribute("data-base") || "";

  var index = null;
  var loading = null;

  // Charge l'index via une balise <script> (search-index.js) injectée à la
  // demande : compatible file:// (contrairement à fetch, bloqué hors http).
  function load() {
    if (index) return Promise.resolve(index);
    if (loading) return loading;
    loading = new Promise(function (resolve) {
      if (window.__SEARCH_INDEX__) {
        index = window.__SEARCH_INDEX__;
        return resolve(index);
      }
      var s = document.createElement("script");
      s.src = base + "search-index.js";
      s.onload = function () { index = window.__SEARCH_INDEX__ || []; resolve(index); };
      s.onerror = function () { index = []; resolve(index); };
      document.head.appendChild(s);
    });
    return loading;
  }

  // Normalise comme le générateur Go : minuscules + diacritiques retirés.
  var DIA = {
    "à": "a", "â": "a", "ä": "a", "á": "a", "ã": "a", "ç": "c",
    "é": "e", "è": "e", "ê": "e", "ë": "e", "î": "i", "ï": "i",
    "í": "i", "ì": "i", "ô": "o", "ö": "o", "ó": "o", "ò": "o",
    "õ": "o", "ù": "u", "û": "u", "ü": "u", "ú": "u", "ÿ": "y",
    "œ": "oe", "æ": "ae", "ñ": "n"
  };
  function norm(s) {
    s = (s || "").toLowerCase();
    var out = "";
    for (var i = 0; i < s.length; i++) {
      var c = s[i];
      out += DIA[c] !== undefined ? DIA[c] : c;
    }
    return out;
  }

  function score(doc, q) {
    var t = norm(doc.title);
    var sec = norm(doc.section || "");
    var body = doc.content || "";
    var s = 0;
    if (t.indexOf(q) >= 0) s += 100;
    if (sec.indexOf(q) >= 0) s += 50;
    if (body.indexOf(q) >= 0) s += 10;
    return s;
  }

  function search(q, limit) {
    q = norm(q.trim());
    if (!q || !index) return [];
    var hits = [];
    for (var i = 0; i < index.length; i++) {
      var s = score(index[i], q);
      if (s > 0) hits.push({ doc: index[i], s: s });
    }
    hits.sort(function (a, b) { return b.s - a.s; });
    return hits.slice(0, limit || 20).map(function (h) { return h.doc; });
  }

  function snippet(content, q) {
    var i = content.indexOf(q);
    if (i < 0) return content.slice(0, 120) + "…";
    var start = Math.max(0, i - 40);
    var end = Math.min(content.length, i + q.length + 80);
    var pre = start > 0 ? "…" : "";
    var post = end < content.length ? "…" : "";
    var seg = content.slice(start, end);
    // Surlignage simple de l'occurrence.
    var rel = seg.toLowerCase().indexOf(q);
    if (rel >= 0) {
      seg = seg.slice(0, rel) + "<mark>" + seg.slice(rel, rel + q.length) + "</mark>" + seg.slice(rel + q.length);
    }
    return pre + seg + post;
  }

  function esc(s) {
    return (s || "").replace(/[&<>"]/g, function (c) {
      return { "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;" }[c];
    });
  }

  function render(results, q, container, asOverlay) {
    if (!results.length) {
      container.innerHTML = '<div class="sr-empty">Aucun résultat</div>';
      return;
    }
    var nq = norm(q);
    var html = results.map(function (d, i) {
      var label = esc(d.title);
      if (d.section) label += ' <span class="sr-sec">› ' + esc(d.section) + "</span>";
      var snip = asOverlay ? "" : '<div class="sr-snippet">' + snippet(d.content, nq) + "</div>";
      return (
        '<a class="sr-item' + (i === 0 ? " sr-active" : "") + '" href="' + base + esc(d.url) + '">' +
        '<div class="sr-title">' + label + "</div>" +
        '<div class="sr-part">' + esc(d.part) + "</div>" +
        snip +
        "</a>"
      );
    }).join("");
    container.innerHTML = html;
  }

  // --- Recherche du header (surimpression) ---
  function initHeader() {
    var input = document.getElementById("q");
    var box = document.getElementById("search-results");
    if (!input || !box) return;

    function update() {
      var q = input.value.trim();
      if (!q) { box.hidden = true; box.innerHTML = ""; return; }
      load().then(function () {
        render(search(q, 8), q, box, true);
        box.hidden = false;
      });
    }

    input.addEventListener("input", update);
    input.addEventListener("focus", function () { if (input.value) update(); });
    document.addEventListener("click", function (e) {
      if (!box.contains(e.target) && e.target !== input) box.hidden = true;
    });

    input.addEventListener("keydown", function (e) {
      var items = box.querySelectorAll(".sr-item");
      var active = box.querySelector(".sr-active");
      var idx = Array.prototype.indexOf.call(items, active);
      if (e.key === "ArrowDown") {
        e.preventDefault();
        if (idx < items.length - 1) move(items, idx, idx + 1);
      } else if (e.key === "ArrowUp") {
        e.preventDefault();
        if (idx > 0) move(items, idx, idx - 1);
      } else if (e.key === "Enter") {
        if (active) { e.preventDefault(); window.location.href = active.getAttribute("href"); }
      } else if (e.key === "Escape") {
        box.hidden = true;
      }
    });

    function move(items, from, to) {
      if (items[from]) items[from].classList.remove("sr-active");
      if (items[to]) {
        items[to].classList.add("sr-active");
        items[to].scrollIntoView({ block: "nearest" });
      }
    }
  }

  // --- Page de recherche dédiée ---
  function initPage() {
    var input = document.getElementById("page-q");
    var out = document.getElementById("page-results");
    if (!input || !out) return;

    function update() {
      var q = input.value.trim();
      if (!q) { out.innerHTML = ""; return; }
      load().then(function () { render(search(q, 40), q, out, false); });
    }
    input.addEventListener("input", update);

    // Pré-remplissage depuis ?q=… (lien depuis le header).
    var params = new URLSearchParams(window.location.search);
    var q0 = params.get("q");
    if (q0) { input.value = q0; update(); }
  }

  // Raccourci clavier « / » : focus sur la recherche du header.
  function initShortcut() {
    document.addEventListener("keydown", function (e) {
      if (e.key === "/" && !/^(INPUT|TEXTAREA)$/.test(document.activeElement.tagName)) {
        var input = document.getElementById("q");
        if (input) { e.preventDefault(); input.focus(); }
      }
    });
  }

  function init() {
    initHeader();
    initPage();
    initShortcut();
  }

  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", init);
  } else {
    init();
  }
})();
