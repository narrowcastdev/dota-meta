(function () {
  "use strict";

  var data = null;
  var currentBracket = "legend_ancient";
  var allHeroesSort = { key: "win_rate", desc: true };

  var bracketSelect = document.getElementById("bracket-select");
  bracketSelect.addEventListener("change", function () {
    currentBracket = bracketSelect.value;
    render();
  });

  fetch("data.json")
    .then(function (response) {
      if (!response.ok) {
        throw new Error("Failed to load data.json: " + response.status);
      }
      return response.json();
    })
    .then(function (json) {
      data = json;
      renderGeneratedTime();
      render();
    })
    .catch(function (err) {
      document.querySelector("main").innerHTML =
        '<p class="empty-state">Failed to load data. ' + err.message + "</p>";
    });

  function render() {
    if (!data) {
      return;
    }
    renderBestHeroes();
    renderSleepers();
    renderTraps();
    renderAllHeroes();
    renderDelta();
  }

  function renderGeneratedTime() {
    var el = document.getElementById("generated-time");
    if (!data || !data.generated) {
      return;
    }
    var date = new Date(data.generated);
    el.textContent = "Last updated: " + date.toLocaleDateString("en-US", {
      year: "numeric",
      month: "long",
      day: "numeric",
      hour: "2-digit",
      minute: "2-digit",
      timeZoneName: "short",
    });
  }

  function renderBestHeroes() {
    var tbody = document.querySelector("#best-table tbody");
    var bracketAnalysis = data.analysis.brackets[currentBracket];
    if (!bracketAnalysis || !bracketAnalysis.best || bracketAnalysis.best.length === 0) {
      tbody.innerHTML = '<tr><td colspan="5" class="empty-state">No data</td></tr>';
      return;
    }

    tbody.innerHTML = bracketAnalysis.best
      .map(function (hero, index) {
        return (
          "<tr>" +
          "<td>" + (index + 1) + "</td>" +
          "<td>" + hero.name + "</td>" +
          "<td class=\"" + wrClass(hero.win_rate) + "\">" + hero.win_rate.toFixed(1) + "%</td>" +
          "<td>" + hero.pick_rate.toFixed(1) + "%</td>" +
          "<td>" + formatNumber(hero.picks) + "</td>" +
          "</tr>"
        );
      })
      .join("");
  }

  function renderSleepers() {
    var tbody = document.querySelector("#sleepers-table tbody");
    var bracketAnalysis = data.analysis.brackets[currentBracket];
    if (!bracketAnalysis || !bracketAnalysis.sleepers || bracketAnalysis.sleepers.length === 0) {
      tbody.innerHTML = '<tr><td colspan="4" class="empty-state">None this week</td></tr>';
      return;
    }

    tbody.innerHTML = bracketAnalysis.sleepers
      .map(function (hero) {
        return (
          "<tr>" +
          "<td>" + hero.name + "</td>" +
          "<td class=\"" + wrClass(hero.win_rate) + "\">" + hero.win_rate.toFixed(1) + "%</td>" +
          "<td>" + hero.pick_rate.toFixed(1) + "%</td>" +
          "<td>" + formatNumber(hero.picks) + "</td>" +
          "</tr>"
        );
      })
      .join("");
  }

  function renderTraps() {
    var tbody = document.querySelector("#traps-table tbody");
    var bracketAnalysis = data.analysis.brackets[currentBracket];
    if (!bracketAnalysis || !bracketAnalysis.traps || bracketAnalysis.traps.length === 0) {
      tbody.innerHTML = '<tr><td colspan="4" class="empty-state">None this week</td></tr>';
      return;
    }

    tbody.innerHTML = bracketAnalysis.traps
      .map(function (hero) {
        return (
          "<tr>" +
          "<td>" + hero.name + "</td>" +
          "<td class=\"" + wrClass(hero.win_rate) + "\">" + hero.win_rate.toFixed(1) + "%</td>" +
          "<td>" + hero.pick_rate.toFixed(1) + "%</td>" +
          "<td>" + formatNumber(hero.picks) + "</td>" +
          "</tr>"
        );
      })
      .join("");
  }

  function renderAllHeroes() {
    var tbody = document.querySelector("#all-heroes-table tbody");
    var heroes = data.heroes
      .map(function (hero) {
        var bracket = hero.brackets[currentBracket];
        if (!bracket) {
          return null;
        }
        return {
          name: hero.name,
          win_rate: bracket.win_rate,
          picks: bracket.picks,
        };
      })
      .filter(function (hero) {
        return hero !== null && hero.picks > 0;
      });

    heroes.sort(function (a, b) {
      var valA = a[allHeroesSort.key];
      var valB = b[allHeroesSort.key];
      if (typeof valA === "string") {
        valA = valA.toLowerCase();
        valB = valB.toLowerCase();
      }
      if (valA < valB) {
        return allHeroesSort.desc ? 1 : -1;
      }
      if (valA > valB) {
        return allHeroesSort.desc ? -1 : 1;
      }
      return 0;
    });

    tbody.innerHTML = heroes
      .map(function (hero) {
        return (
          "<tr>" +
          "<td>" + hero.name + "</td>" +
          "<td class=\"" + wrClass(hero.win_rate) + "\">" + hero.win_rate.toFixed(1) + "%</td>" +
          "<td>" + formatNumber(hero.picks) + "</td>" +
          "</tr>"
        );
      })
      .join("");

    // Update sort indicators
    var headers = document.querySelectorAll("#all-heroes-table th.sortable");
    for (var i = 0; i < headers.length; i++) {
      headers[i].classList.remove("sort-asc", "sort-desc");
      if (headers[i].getAttribute("data-sort") === allHeroesSort.key) {
        headers[i].classList.add(allHeroesSort.desc ? "sort-desc" : "sort-asc");
      }
    }
  }

  // Sort click handlers
  document.querySelectorAll("#all-heroes-table th.sortable").forEach(function (th) {
    th.addEventListener("click", function () {
      var key = th.getAttribute("data-sort");
      if (allHeroesSort.key === key) {
        allHeroesSort.desc = !allHeroesSort.desc;
      } else {
        allHeroesSort.key = key;
        allHeroesSort.desc = key !== "name";
      }
      renderAllHeroes();
    });
  });

  function renderDelta() {
    renderDeltaTable(
      "#low-stompers-table tbody",
      data.analysis.delta.low_stompers,
      true
    );
    renderDeltaTable(
      "#high-ceiling-table tbody",
      data.analysis.delta.high_skill_ceiling,
      false
    );
  }

  function renderDeltaTable(selector, heroes, isLowStomper) {
    var tbody = document.querySelector(selector);
    if (!heroes || heroes.length === 0) {
      tbody.innerHTML = '<tr><td colspan="4" class="empty-state">None this week</td></tr>';
      return;
    }

    tbody.innerHTML = heroes
      .map(function (hero) {
        var diff = Math.abs(hero.delta);
        var diffClass = isLowStomper ? "delta-positive" : "delta-negative";
        return (
          "<tr>" +
          "<td>" + hero.name + "</td>" +
          "<td class=\"" + wrClass(hero.low_wr) + "\">" + hero.low_wr.toFixed(1) + "%</td>" +
          "<td class=\"" + wrClass(hero.high_wr) + "\">" + hero.high_wr.toFixed(1) + "%</td>" +
          "<td class=\"" + diffClass + "\">" + diff.toFixed(1) + "%</td>" +
          "</tr>"
        );
      })
      .join("");
  }

  function wrClass(wr) {
    if (wr >= 53) {
      return "wr-high";
    }
    if (wr < 48) {
      return "wr-low";
    }
    return "wr-neutral";
  }

  function formatNumber(n) {
    return n.toString().replace(/\B(?=(\d{3})+(?!\d))/g, ",");
  }
})();
