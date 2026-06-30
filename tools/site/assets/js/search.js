// search.js — recherche plein-texte côté client, sans dépendance.
//
// Charge search-index.json au premier usage, filtre par sous-chaîne normalisée
// (insensible aux accents et à la casse), classe les résultats (titre > section
// > corps) et les affiche : en surimpression depuis le header, et en liste pleine
// page sur search.html. Navigation clavier ↑/↓/Entrée, focus rapide avec « / ».
(() => {
  let base = "";
  const script = document.querySelector('script[src*="search.js"]');
  if (script) base = script.getAttribute("data-base") || "";

  let index = null;
  let loading = null;

  // Charge l'index via une balise <script> (search-index.js) injectée à la
  // demande : compatible file:// (contrairement à fetch, bloqué hors http).
  const load = () => {
    if (index) return Promise.resolve(index);
    if (loading) return loading;
    loading = new Promise((resolve) => {
      if (window.__SEARCH_INDEX__) {
        index = window.__SEARCH_INDEX__;
        return resolve(index);
      }
      const s = document.createElement("script");
      s.src = `${base}search-index.js`;
      s.onload = () => {
        index = window.__SEARCH_INDEX__ || [];
        resolve(index);
      };
      s.onerror = () => {
        index = [];
        resolve(index);
      };
      document.head.appendChild(s);
    });
    return loading;
  };

  // Normalise comme le générateur Go : minuscules + diacritiques retirés.
  const DIA = {
    "à": "a", "â": "a", "ä": "a", "á": "a", "ã": "a", "ç": "c",
    "é": "e", "è": "e", "ê": "e", "ë": "e", "î": "i", "ï": "i",
    "í": "i", "ì": "i", "ô": "o", "ö": "o", "ó": "o", "ò": "o",
    "õ": "o", "ù": "u", "û": "u", "ü": "u", "ú": "u", "ÿ": "y",
    "œ": "oe", "æ": "ae", "ñ": "n",
  };
  const norm = (s) => {
    const lower = (s || "").toLowerCase();
    let out = "";
    for (let i = 0; i < lower.length; i++) {
      const c = lower[i];
      out += DIA[c] !== undefined ? DIA[c] : c;
    }
    return out;
  };

  const score = (doc, q) => {
    const t = norm(doc.title);
    const sec = norm(doc.section || "");
    const body = doc.content || "";
    let s = 0;
    if (t.indexOf(q) >= 0) s += 100;
    if (sec.indexOf(q) >= 0) s += 50;
    if (body.indexOf(q) >= 0) s += 10;
    return s;
  };

  const search = (q, limit) => {
    const nq = norm(q.trim());
    if (!nq || !index) return [];
    const hits = [];
    for (let i = 0; i < index.length; i++) {
      const s = score(index[i], nq);
      if (s > 0) hits.push({ doc: index[i], s: s });
    }
    hits.sort((a, b) => b.s - a.s);
    return hits.slice(0, limit || 20).map((h) => h.doc);
  };

  const snippet = (content, q) => {
    const i = content.indexOf(q);
    if (i < 0) return `${content.slice(0, 120)}…`;
    const start = Math.max(0, i - 40);
    const end = Math.min(content.length, i + q.length + 80);
    const pre = start > 0 ? "…" : "";
    const post = end < content.length ? "…" : "";
    let seg = content.slice(start, end);
    // Surlignage simple de l'occurrence.
    const rel = seg.toLowerCase().indexOf(q);
    if (rel >= 0) {
      seg =
        seg.slice(0, rel) +
        "<mark>" +
        seg.slice(rel, rel + q.length) +
        "</mark>" +
        seg.slice(rel + q.length);
    }
    return pre + seg + post;
  };

  const esc = (s) =>
    (s || "").replace(
      /[&<>"]/g,
      (c) => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;" })[c],
    );

  const render = (results, q, container, asOverlay) => {
    if (!results.length) {
      container.innerHTML = '<div class="sr-empty">Aucun résultat</div>';
      return;
    }
    const nq = norm(q);
    container.innerHTML = results
      .map((d, i) => {
        let label = esc(d.title);
        if (d.section) label += ` <span class="sr-sec">› ${esc(d.section)}</span>`;
        const snip = asOverlay
          ? ""
          : `<div class="sr-snippet">${snippet(d.content, nq)}</div>`;
        return (
          `<a class="sr-item${i === 0 ? " sr-active" : ""}" href="${base}${esc(d.url)}">` +
          `<div class="sr-title">${label}</div>` +
          `<div class="sr-part">${esc(d.part)}</div>` +
          snip +
          "</a>"
        );
      })
      .join("");
  };

  // --- Recherche du header (surimpression) ---
  const initHeader = () => {
    const input = document.getElementById("q");
    const box = document.getElementById("search-results");
    if (!input || !box) return;

    const update = () => {
      const q = input.value.trim();
      if (!q) {
        box.hidden = true;
        box.innerHTML = "";
        return;
      }
      load().then(() => {
        render(search(q, 8), q, box, true);
        box.hidden = false;
      });
    };

    const move = (items, from, to) => {
      if (items[from]) items[from].classList.remove("sr-active");
      if (items[to]) {
        items[to].classList.add("sr-active");
        items[to].scrollIntoView({ block: "nearest" });
      }
    };

    input.addEventListener("input", update);
    input.addEventListener("focus", () => {
      if (input.value) update();
    });
    document.addEventListener("click", (e) => {
      if (!box.contains(e.target) && e.target !== input) box.hidden = true;
    });

    input.addEventListener("keydown", (e) => {
      const items = box.querySelectorAll(".sr-item");
      const active = box.querySelector(".sr-active");
      const idx = Array.prototype.indexOf.call(items, active);
      if (e.key === "ArrowDown") {
        e.preventDefault();
        if (idx < items.length - 1) move(items, idx, idx + 1);
      } else if (e.key === "ArrowUp") {
        e.preventDefault();
        if (idx > 0) move(items, idx, idx - 1);
      } else if (e.key === "Enter") {
        if (active) {
          e.preventDefault();
          window.location.href = active.getAttribute("href");
        }
      } else if (e.key === "Escape") {
        box.hidden = true;
      }
    });
  };

  // --- Page de recherche dédiée ---
  const initPage = () => {
    const input = document.getElementById("page-q");
    const out = document.getElementById("page-results");
    if (!input || !out) return;

    // Place le curseur dans le champ à l'ouverture de la page (équivaut à
    // l'attribut autofocus, mais sans le signalement d'accessibilité associé).
    input.focus();

    const update = () => {
      const q = input.value.trim();
      if (!q) {
        out.innerHTML = "";
        return;
      }
      load().then(() => render(search(q, 40), q, out, false));
    };
    input.addEventListener("input", update);

    // Pré-remplissage depuis ?q=… (lien depuis le header).
    const params = new URLSearchParams(window.location.search);
    const q0 = params.get("q");
    if (q0) {
      input.value = q0;
      update();
    }
  };

  // Raccourci clavier « / » : focus sur la recherche du header.
  const initShortcut = () => {
    document.addEventListener("keydown", (e) => {
      if (
        e.key === "/" &&
        !/^(INPUT|TEXTAREA)$/.test(document.activeElement.tagName)
      ) {
        const input = document.getElementById("q");
        if (input) {
          e.preventDefault();
          input.focus();
        }
      }
    });
  };

  const init = () => {
    initHeader();
    initPage();
    initShortcut();
  };

  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", init);
  } else {
    init();
  }
})();
