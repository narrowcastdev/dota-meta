(function () {
  "use strict";

  var MOMENTUM_ICON = { rising: "🔥", "falling-off": "⚠️", "hidden-gem": "💤", dying: "📉" };
  var CLIMB_ICON = { "scales-up": "↗", "scales-down": "↘", universal: "≈" };
  var TIER_ORDER = ["meta-tyrant", "pocket-pick", "trap", "dead"];
  var TIER_TITLE = {
    "meta-tyrant": "Meta Tyrant — ban or first-pick",
    "pocket-pick": "Pocket Pick — last-pick counter",
    trap:          "Trap — popular but losing",
    dead:          "Dead — ignore",
  };
  var BRACKET_KEYS = ["herald_guardian", "crusader_archon", "legend_ancient", "divine", "immortal"];

  var data = null;
  var currentBracket = localStorage.getItem("dotaMeta_bracket") || "divine";
  var currentRole = localStorage.getItem("dotaMeta_role") || "all";

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
      render();
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
    var rows = TIER_ORDER.map(function (t) {
      return '<span class="legend-tier tier-' + t + '">' + TIER_TITLE[t] + "</span>";
    }).join("");

    var climbRows = Object.keys(CLIMB_ICON).map(function (k) {
      return "<span>" + CLIMB_ICON[k] + " " + k + "</span>";
    }).join(" &nbsp;");

    var momRows = Object.keys(MOMENTUM_ICON).map(function (k) {
      return "<span>" + MOMENTUM_ICON[k] + " " + k + "</span>";
    }).join(" &nbsp;");

    el.innerHTML =
      '<div class="legend">' +
      '<div class="legend-row"><strong>Tiers:</strong> ' + rows + "</div>" +
      '<div class="legend-row"><strong>Climb:</strong> ' + climbRows + "</div>" +
      '<div class="legend-row"><strong>Momentum:</strong> ' + momRows + "</div>" +
      "</div>";
  }

  function render() {
    if (!data) return;
    var tiersEl = document.getElementById("tiers");
    tiersEl.innerHTML = "";

    var brackets = data.analysis.brackets;
    var bracketData = brackets[currentBracket];
    if (!bracketData) {
      tiersEl.innerHTML = '<p class="empty-state">No data for this bracket.</p>';
      return;
    }

    if (currentRole === "all" || currentRole === "cores") {
      tiersEl.appendChild(renderRoleSection("Cores", bracketData.cores || []));
    }
    if (currentRole === "all" || currentRole === "supports") {
      tiersEl.appendChild(renderRoleSection("Supports", bracketData.supports || []));
    }
  }

  function renderRoleSection(roleLabel, heroes) {
    var section = document.createElement("section");
    section.className = "role-section";

    var heading = document.createElement("h2");
    heading.textContent = roleLabel;
    section.appendChild(heading);

    var grid = document.createElement("div");
    grid.className = "tier-grid";

    TIER_ORDER.forEach(function (tier) {
      var heroesInTier = heroes
        .filter(function (h) { return h.tier === tier; })
        .sort(function (a, b) { return b.win_rate - a.win_rate; });

      var cell = document.createElement("div");
      cell.className = "tier-cell tier-" + tier + (tier === "dead" ? " collapsed" : "");

      var header = document.createElement("div");
      header.className = "tier-cell-header";
      header.textContent = TIER_TITLE[tier] + " (" + heroesInTier.length + ")";
      header.addEventListener("click", function () {
        cell.classList.toggle("collapsed");
      });

      var chips = document.createElement("div");
      chips.className = "chips";
      chips.innerHTML = heroesInTier.map(renderChip).join("");

      cell.appendChild(header);
      cell.appendChild(chips);
      grid.appendChild(cell);
    });

    section.appendChild(grid);
    return section;
  }

  function renderChip(hero) {
    var parts = [];
    parts.push('<span class="chip-name">' + hero.name + "</span>");

    if (hero.climb && CLIMB_ICON[hero.climb]) {
      parts.push('<span class="chip-climb" title="' + hero.climb + '">' + CLIMB_ICON[hero.climb] + "</span>");
    }

    var wrStr = hero.win_rate.toFixed(1) + "%";
    if (hero.wr_delta != null && hero.wr_delta !== 0) {
      var wrSign = hero.wr_delta > 0 ? "+" : "";
      var wrClass = hero.wr_delta > 0 ? "up" : "down";
      wrStr += ' <span class="delta ' + wrClass + '">' + wrSign + hero.wr_delta.toFixed(1) + "</span>";
    }
    parts.push('<span class="chip-wr">WR ' + wrStr + "</span>");

    var prStr = hero.pick_rate.toFixed(1) + "%";
    if (hero.pr_delta != null && hero.pr_delta !== 0) {
      var prSign = hero.pr_delta > 0 ? "+" : "";
      var prClass = hero.pr_delta > 0 ? "up" : "down";
      prStr += ' <span class="delta ' + prClass + '">' + prSign + hero.pr_delta.toFixed(1) + "</span>";
    }
    parts.push('<span class="chip-pr">PR ' + prStr + "</span>");

    if (hero.momentum && MOMENTUM_ICON[hero.momentum]) {
      parts.push('<span class="chip-momentum" title="' + hero.momentum + '">' + MOMENTUM_ICON[hero.momentum] + "</span>");
    }

    var sparkline = "";
    if (hero.wr_history && hero.wr_history.length > 1) {
      sparkline = '<span class="sparkline">' + buildSparkline(hero.wr_history) + "</span>";
    }

    return '<span class="chip">' + parts.join("") + sparkline + "</span>";
  }

  function buildSparkline(history) {
    var w = 80;
    var h = 24;
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
      '<svg width="' + w + '" height="' + h + '" viewBox="0 0 ' + w + " " + h + '">' +
      '<polyline points="' + points + '" fill="none" stroke="#60a5fa" stroke-width="1.5"/>' +
      "</svg>"
    );
  }

})();
