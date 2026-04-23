(function () {
  "use strict";

  var MOMENTUM_ICON = {
    rising: "🔥",
    "falling-off": "⚠️",
    "hidden-gem": "💎",
    dying: "📉",
  };
  var CLIMB_ICON = {
    "high-skill": "🧠",
    pubstomp: "🪓",
    "all-rank": "⚖️",
  };
  var TIER_ICON = {
    "meta-tyrant": "👑",
    "pocket-pick": "🎯",
    trap: "🪤",
    dead: "💀",
  };
  var TIER_ORDER = ["meta-tyrant", "pocket-pick", "trap", "dead"];
  var TIER_TITLE = {
    "meta-tyrant": "Meta Tyrants — ban or first-pick",
    "pocket-pick": "Pocket Picks — last-pick counters",
    trap: "Traps — popular but losing",
    dead: "Dead — ignore",
  };
  var BRACKET_KEYS = ["herald_guardian", "crusader_archon", "legend_ancient", "divine", "immortal"];

  var TIER_SLUG = {
    "meta-tyrant": "meta-tyrants",
    "pocket-pick": "pocket-picks",
    trap: "traps",
    dead: "dead",
  };

  var data = null;
  var hashState = parseHash();
  var currentBracket =
    hashState.bracket ||
    localStorage.getItem("dotaMeta_bracket") ||
    "divine";
  var currentRole = localStorage.getItem("dotaMeta_role") || "all";

  function parseHash() {
    var h = (location.hash || "").replace(/^#/, "");
    if (!h) return {};
    var parts = h.split("-");
    // Bracket keys like "herald_guardian", "crusader_archon" use underscores;
    // also try the first token (e.g. "divine", "immortal").
    for (var i = 0; i < BRACKET_KEYS.length; i++) {
      var key = BRACKET_KEYS[i];
      if (h === key) return { bracket: key };
      if (h.indexOf(key + "-") === 0) {
        return { bracket: key, anchor: h };
      }
    }
    return {};
  }

  function updateHash(bracket) {
    if (history.replaceState) {
      history.replaceState(null, "", "#" + bracket);
    } else {
      location.hash = bracket;
    }
  }

  fetch("data.json")
    .then(function (r) {
      if (!r.ok) throw new Error("HTTP " + r.status);
      return r.json();
    })
    .then(function (json) {
      data = json;
      initControls();
      renderHeader();
      renderLegend();
      render();
    })
    .catch(function (err) {
      document.querySelector("main").innerHTML =
        '<p class="empty-state">Failed to load data. ' + err.message + "</p>";
    });

  function initControls() {
    var bracketEl = document.getElementById("bracket");
    var brackets = data.analysis.brackets;
    BRACKET_KEYS.forEach(function (key) {
      if (!brackets[key]) return;
      var opt = document.createElement("option");
      opt.value = key;
      opt.textContent = brackets[key].name;
      if (key === currentBracket) opt.selected = true;
      bracketEl.appendChild(opt);
    });
    bracketEl.addEventListener("change", function () {
      currentBracket = bracketEl.value;
      localStorage.setItem("dotaMeta_bracket", currentBracket);
      updateHash(currentBracket);
      render();
    });

    window.addEventListener("hashchange", function () {
      var st = parseHash();
      if (st.bracket && st.bracket !== currentBracket) {
        currentBracket = st.bracket;
        localStorage.setItem("dotaMeta_bracket", currentBracket);
        for (var j = 0; j < bracketEl.options.length; j++) {
          if (bracketEl.options[j].value === currentBracket) {
            bracketEl.options[j].selected = true;
            break;
          }
        }
        render();
      }
      if (st.anchor) scrollToAnchor(st.anchor);
    });

    var roleEl = document.getElementById("role");
    for (var i = 0; i < roleEl.options.length; i++) {
      if (roleEl.options[i].value === currentRole) {
        roleEl.options[i].selected = true;
        break;
      }
    }
    roleEl.addEventListener("change", function () {
      currentRole = roleEl.value;
      localStorage.setItem("dotaMeta_role", currentRole);
      render();
    });
  }

  function renderHeader() {
    var meta = document.getElementById("header-meta");
    var line = data.patch + " · snapshot " + data.snapshot_date;
    if (data.prior_snapshot) line += " · vs " + data.prior_snapshot;
    meta.textContent = line;

    var genEl = document.getElementById("generated-time");
    if (data.generated) {
      var d = new Date(data.generated);
      genEl.textContent = "Generated: " + d.toLocaleString("en-US", {
        year: "numeric", month: "short", day: "numeric",
        hour: "2-digit", minute: "2-digit", timeZoneName: "short",
      });
    }
  }

  function renderLegend() {
    var el = document.getElementById("legend");

    var tiers = TIER_ORDER.map(function (t) {
      return '<span class="legend-tier tier-' + t + '">' +
        TIER_ICON[t] + " " + TIER_TITLE[t] + "</span>";
    }).join("");


    var mom = Object.keys(MOMENTUM_ICON).map(function (k) {
      return "<span>" + MOMENTUM_ICON[k] + " " + k + "</span>";
    }).join(" &nbsp;");

    var trend =
      '<span>↗ rising WR</span> &nbsp;' +
      '<span>→ flat</span> &nbsp;' +
      '<span>↘ falling WR</span>';

    el.innerHTML =
      '<div class="legend">' +
      '<div class="legend-row"><strong>Tiers:</strong> ' + tiers + "</div>" +
      '<div class="legend-row"><strong>Momentum:</strong> ' + mom + "</div>" +
      '<div class="legend-row"><strong>Trend:</strong> ' + trend + "</div>" +
      "</div>";
  }

  function render() {
    if (!data) return;
    var root = document.getElementById("tiers");
    root.innerHTML = "";

    var bracketData = data.analysis.brackets[currentBracket];
    if (!bracketData) {
      root.innerHTML = '<p class="empty-state">No data for this bracket.</p>';
      return;
    }

    var showCores = currentRole === "all" || currentRole === "cores";
    var showSupports = currentRole === "all" || currentRole === "supports";

    // Part 1 — full ranking tables (old format).
    if (showCores) {
      var coresSection = renderRankingSection("Cores — full ranking", bracketData.cores || []);
      coresSection.id = currentBracket + "-cores";
      root.appendChild(coresSection);
    }
    if (showSupports) {
      var suppSection = renderRankingSection("Supports — full ranking", bracketData.supports || []);
      suppSection.id = currentBracket + "-supports";
      root.appendChild(suppSection);
    }

    // Part 2 — analysis broken out by tier.
    var combined = [];
    if (showCores) combined = combined.concat((bracketData.cores || []).map(tag("core")));
    if (showSupports) combined = combined.concat((bracketData.supports || []).map(tag("support")));

    var analysisWrap = document.createElement("section");
    analysisWrap.className = "analysis-wrap";
    analysisWrap.id = currentBracket + "-analysis";
    var title = document.createElement("h2");
    title.textContent = "Analysis";
    analysisWrap.appendChild(title);

    TIER_ORDER.forEach(function (tier) {
      var heroes = combined.filter(function (h) { return h.tier === tier; });
      var sect = renderTierSection(tier, heroes);
      sect.id = currentBracket + "-" + TIER_SLUG[tier];
      analysisWrap.appendChild(sect);
    });

    root.appendChild(analysisWrap);

    if (hashState.anchor) {
      scrollToAnchor(hashState.anchor);
      hashState.anchor = null;
    }
  }

  function scrollToAnchor(id) {
    var el = document.getElementById(id);
    if (el && el.scrollIntoView) {
      el.scrollIntoView({ behavior: "smooth", block: "start" });
    }
  }

  function tag(role) {
    return function (h) {
      var clone = {};
      for (var k in h) clone[k] = h[k];
      clone._role = role;
      return clone;
    };
  }

  // ── Ranking table (old per-bracket WR table) ──

  function renderRankingSection(label, heroes) {
    var section = document.createElement("section");
    section.className = "ranking-section";

    var h2 = document.createElement("h2");
    h2.textContent = label + " (" + heroes.length + ")";
    section.appendChild(h2);

    var sorted = heroes.slice().sort(function (a, b) { return b.win_rate - a.win_rate; });
    section.appendChild(buildTable(sorted, { rank: true }));
    return section;
  }

  // ── Tier analysis table ──

  function renderTierSection(tier, heroes) {
    var section = document.createElement("section");
    section.className = "tier-section tier-" + tier + (tier === "dead" ? " collapsed" : "");

    var header = document.createElement("h3");
    header.className = "tier-section-header";
    header.innerHTML = TIER_ICON[tier] + " " + TIER_TITLE[tier] +
      ' <span class="count">(' + heroes.length + ")</span>";
    header.addEventListener("click", function () {
      section.classList.toggle("collapsed");
    });
    section.appendChild(header);

    if (heroes.length === 0) {
      var empty = document.createElement("p");
      empty.className = "empty-state";
      empty.textContent = "none";
      section.appendChild(empty);
      return section;
    }

    var sortKey = (tier === "trap") ? "pick_rate" : "win_rate";
    var sorted = heroes.slice().sort(function (a, b) { return b[sortKey] - a[sortKey]; });
    section.appendChild(buildTable(sorted, { rank: false, role: true }));
    return section;
  }

  // ── Shared table builder ──

  function buildTable(heroes, opts) {
    opts = opts || {};
    var table = document.createElement("table");
    table.className = "hero-table";

    var thead = document.createElement("thead");
    var headCells = [];
    if (opts.rank) headCells.push("#");
    headCells.push("Hero");
    if (opts.role) headCells.push("Role");
    headCells.push("Tier");
    headCells.push("WR%");
    headCells.push("ΔWR");
    headCells.push("PR%");
    headCells.push("ΔPR");
    headCells.push("Picks");
    headCells.push("Momentum");
    headCells.push("Trend");

    var trh = document.createElement("tr");
    headCells.forEach(function (h) {
      var th = document.createElement("th");
      th.textContent = h;
      trh.appendChild(th);
    });
    thead.appendChild(trh);
    table.appendChild(thead);

    var tbody = document.createElement("tbody");
    heroes.forEach(function (hero, idx) {
      tbody.appendChild(renderRow(hero, idx + 1, opts));
    });
    table.appendChild(tbody);

    return table;
  }

  function renderRow(hero, rank, opts) {
    var tr = document.createElement("tr");
    tr.className = "tier-row tier-" + hero.tier;

    var cells = [];
    if (opts.rank) cells.push(td(String(rank), "num"));
    cells.push(td(hero.name, "name"));
    if (opts.role) cells.push(td(hero._role || "", "role"));

    cells.push(tdHTML(
      '<span class="tier-badge tier-' + hero.tier + '" title="' + hero.tier + '">' +
        TIER_ICON[hero.tier] + "</span>",
      "tier"
    ));

    cells.push(td(hero.win_rate.toFixed(1), "num wr"));
    cells.push(tdHTML(formatDelta(hero.wr_delta), "num delta-cell"));
    cells.push(td(hero.pick_rate.toFixed(1), "num pr"));
    cells.push(tdHTML(formatDelta(hero.pr_delta), "num delta-cell"));
    cells.push(td(formatPicks(hero.picks), "num picks"));


    cells.push(tdHTML(
      hero.momentum && MOMENTUM_ICON[hero.momentum]
        ? '<span title="' + hero.momentum + '">' + MOMENTUM_ICON[hero.momentum] + "</span>"
        : '<span class="muted">—</span>',
      "momentum"
    ));

    cells.push(tdHTML(renderTrend(hero.wr_history), "trend"));

    cells.forEach(function (c) { tr.appendChild(c); });
    return tr;
  }

  function td(text, cls) {
    var el = document.createElement("td");
    el.className = cls || "";
    el.textContent = text;
    return el;
  }

  function tdHTML(html, cls) {
    var el = document.createElement("td");
    el.className = cls || "";
    el.innerHTML = html;
    return el;
  }

  function formatDelta(v) {
    if (v == null) return '<span class="muted">—</span>';
    if (Math.abs(v) < 0.05) return '<span class="muted">→ 0.0</span>';
    var sign = v > 0 ? "+" : "";
    var arrow = v > 0 ? "▲" : "▼";
    var cls = v > 0 ? "up" : "down";
    return '<span class="delta ' + cls + '">' + arrow + " " + sign + v.toFixed(1) + "</span>";
  }

  function formatPicks(n) {
    if (n == null) return "";
    if (n >= 1000) return (n / 1000).toFixed(n >= 10000 ? 0 : 1) + "k";
    return String(n);
  }

  function renderTrend(history) {
    if (!history || history.length < 2) return '<span class="muted">—</span>';
    var slope = recentSlope(history);
    var arrow, cls;
    if (slope > 0.3) { arrow = "↗"; cls = "up"; }
    else if (slope < -0.3) { arrow = "↘"; cls = "down"; }
    else { arrow = "→"; cls = "flat"; }
    return '<span class="trend ' + cls + '" title="recent WR slope ' + slope.toFixed(2) + 'pp/wk">' +
      arrow + " " + sparklineSVG(history) + "</span>";
  }

  function recentSlope(history) {
    var n = Math.min(4, history.length);
    var y = history.slice(history.length - n);
    var sx = 0, sy = 0, sxy = 0, sxx = 0;
    for (var i = 0; i < n; i++) {
      sx += i; sy += y[i]; sxy += i * y[i]; sxx += i * i;
    }
    var denom = n * sxx - sx * sx;
    if (denom === 0) return 0;
    return (n * sxy - sx * sy) / denom;
  }

  function sparklineSVG(history) {
    var w = 60, h = 16;
    var n = history.length;
    var min = Math.min.apply(null, history);
    var max = Math.max.apply(null, history);
    var range = max - min || 1;
    var points = history.map(function (wr, i) {
      var x = (i / (n - 1)) * w;
      var y = h - ((wr - min) / range) * h;
      return x.toFixed(1) + "," + y.toFixed(1);
    }).join(" ");
    return (
      '<svg class="spark" width="' + w + '" height="' + h + '" viewBox="0 0 ' + w + " " + h + '">' +
      '<polyline points="' + points + '" fill="none" stroke="currentColor" stroke-width="1.2"/>' +
      "</svg>"
    );
  }
})();
