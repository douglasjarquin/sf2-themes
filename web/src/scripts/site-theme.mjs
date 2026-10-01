const STORAGE_KEY = "sf2-site-theme";
const UVX = "uvx --from git+https://github.com/douglasjarquin/sf2-themes.git sf2-themes";
const root = document.documentElement;

function readFamilies() {
  const el = document.getElementById("site-theme-data");
  if (!el) return [];
  try {
    const parsed = JSON.parse(el.textContent || "null");
    return parsed && Array.isArray(parsed.families) ? parsed.families : [];
  } catch {
    return [];
  }
}

const families = readFamilies();

if (families.length > 0) {
  const byId = (id) => families.find((f) => f.id === id);
  let state = {
    id: byId(root.dataset.theme)?.id ?? byId("ryu")?.id ?? families[0].id,
    mode: root.dataset.mode === "light" ? "light" : "dark",
  };

  const family = () => byId(state.id) ?? byId("ryu") ?? families[0];
  const palette = () => family()[state.mode] ?? family().dark;
  const catalogId = () => (state.mode === "light" ? `${family().id}-light` : family().id);
  const displayName = () => family().name + (state.mode === "light" ? " Light" : "");

  const select = document.querySelector("[data-site-select]");
  const modeButtons = [...document.querySelectorAll("[data-site-mode]")];
  const trigger = document.querySelector("[data-palette-trigger]");
  const backdrop = document.querySelector("[data-palette-backdrop]");
  const input = document.querySelector("[data-palette-input]");
  const list = document.querySelector("[data-palette-list]");
  const empty = document.querySelector("[data-palette-empty]");
  const items = list ? [...list.querySelectorAll("[data-palette-item]")] : [];
  const toggleLabel = document.querySelector("[data-mode-toggle-label]");
  const tBindings = [...document.querySelectorAll("[data-t]")];
  const portButtons = [...document.querySelectorAll("[data-port-id]")];
  const portNote = document.querySelector("[data-port-note-out]");
  const steps = [...document.querySelectorAll("[data-step]")];

  let paletteOpen = false;
  let query = "";
  let sel = 0;
  let port = portButtons[0]?.dataset.portId ?? "wezterm";
  const copyTimers = new Map();

  function applyVars() {
    const v = palette();
    const vars = {
      "--bg": v.bg,
      "--surface": v.surface,
      "--border": v.border,
      "--line": v.border,
      "--fg": v.fg,
      "--muted": v.muted,
      "--subtle": v.subtle,
      "--accent": v.accent,
      "--accent2": v.accent2,
      "--sel": v.sel,
      "--orange": v.orange,
    };
    v.n.forEach((color, i) => { vars[`--c${i}`] = color; });
    v.b.forEach((color, i) => { vars[`--c${i + 8}`] = color; });
    for (const [key, value] of Object.entries(vars)) root.style.setProperty(key, value);
    root.style.colorScheme = state.mode;
    root.dataset.theme = family().id;
    root.dataset.mode = state.mode;
  }

  function persist() {
    try {
      localStorage.setItem(STORAGE_KEY, JSON.stringify(state));
    } catch {}
    const url = new URL(window.location.href);
    url.searchParams.set("theme", state.id);
    url.searchParams.set("mode", state.mode);
    history.replaceState(null, "", url);
  }

  function setTheme(id, mode = state.mode) {
    if (!byId(id)) return;
    state = { id, mode: mode === "light" ? "light" : "dark" };
    applyVars();
    persist();
    syncControls();
    rewritePage();
  }

  function step(d) {
    const index = families.findIndex((f) => f.id === state.id);
    setTheme(families[(index + d + families.length) % families.length].id);
  }

  const toggleMode = () => setTheme(state.id, state.mode === "dark" ? "light" : "dark");

  const visibleItems = () => {
    const words = query.trim().toLowerCase().split(/\s+/).filter(Boolean);
    return items.filter(
      (it) => words.length === 0 || words.every((w) => (it.dataset.hay ?? "").includes(w)),
    );
  };

  function renderPalette() {
    const vis = visibleItems();
    const selItem = vis[Math.min(sel, vis.length - 1)];
    for (const it of items) {
      it.hidden = !vis.includes(it);
      it.toggleAttribute("data-selected", it === selItem);
    }
    if (empty) empty.hidden = vis.length > 0;
  }

  function openPalette() {
    if (!backdrop) return;
    paletteOpen = true;
    backdrop.hidden = false;
    query = "";
    if (input) input.value = "";
    sel = 0;
    renderPalette();
    syncControls();
    input?.focus();
  }

  function closePalette(restore = true) {
    paletteOpen = false;
    if (backdrop) backdrop.hidden = true;
    syncControls();
    if (restore) trigger?.focus();
  }

  function runItem(item) {
    if (!item) return;
    if (item.hasAttribute("data-mode-toggle")) {
      toggleMode();
    } else {
      setTheme(item.dataset.themeId, item.dataset.themeMode);
    }
    closePalette(false);
  }

  function tValue(key) {
    const f = family();
    const v = palette();
    switch (key) {
      case "name": return f.name;
      case "displayName": return displayName();
      case "catalogId": return catalogId();
      case "id": return f.id;
      case "fileId": return `sf2-${catalogId()}`;
      case "stage": return f.stage;
      case "mode": return state.mode;
      default: break;
    }
    if (key in v && typeof v[key] === "string") return v[key];
    const ramp = /^([nb])([0-7])$/.exec(key);
    return ramp ? v[ramp[1]][Number(ramp[2])] : null;
  }

  function stepCommands() {
    const p = port;
    const name = portButtons.find((b) => b.dataset.portId === p)?.dataset.portName ?? p;
    return [
      { label: "Get the CLI", display: "sf2-themes --version", cmd: `${UVX} --version` },
      { label: `Set up ${name}, once`, display: `sf2-themes setup ${p}`, cmd: `${UVX} setup ${p}` },
      {
        label: `Apply ${displayName()}. K.O.`,
        display: `sf2-themes apply ${p} --theme ${catalogId()}`,
        cmd: `${UVX} apply ${p} --theme ${catalogId()}`,
      },
    ];
  }

  function updateSteps() {
    if (!steps.length) return;
    const cmds = stepCommands();
    steps.forEach((el, i) => {
      const cmd = cmds[i];
      if (!cmd) return;
      const label = el.querySelector("[data-step-label]");
      const display = el.querySelector("[data-step-cmd]");
      const copy = el.querySelector("[data-step-copy]");
      if (label) label.textContent = cmd.label;
      if (display) display.textContent = cmd.display;
      if (copy) copy.dataset.copyText = cmd.cmd;
    });
    if (portNote) {
      const note = portButtons.find((b) => b.dataset.portId === port)?.dataset.portNote;
      if (note) portNote.textContent = `${note} Copied commands include the uvx prefix.`;
    }
  }

  function rewritePage() {
    for (const el of tBindings) {
      const value = tValue(el.dataset.t);
      if (value != null) el.textContent = value;
    }
    updateSteps();
  }

  function syncControls() {
    if (select) select.value = state.id;
    for (const b of modeButtons) {
      b.setAttribute("aria-pressed", String(b.dataset.siteMode === state.mode));
    }
    trigger?.setAttribute("aria-expanded", String(paletteOpen));
    for (const it of items) {
      const current = it.dataset.themeId === state.id && it.dataset.themeMode === state.mode;
      it.toggleAttribute("data-current", current);
      it.setAttribute("aria-selected", String(current));
    }
    if (toggleLabel) {
      toggleLabel.textContent = `Switch to ${state.mode === "dark" ? "light" : "dark"} mode`;
    }
  }

  select?.addEventListener("change", () => setTheme(select.value));
  for (const b of modeButtons) {
    b.addEventListener("click", () => setTheme(state.id, b.dataset.siteMode));
  }
  trigger?.addEventListener("click", openPalette);
  backdrop?.addEventListener("mousedown", (e) => {
    if (e.target === backdrop) closePalette(false);
  });
  input?.addEventListener("input", () => {
    query = input.value;
    sel = 0;
    renderPalette();
  });
  input?.addEventListener("keydown", (e) => {
    const vis = visibleItems();
    if (e.key === "ArrowDown") {
      e.preventDefault();
      sel = Math.min(sel + 1, vis.length - 1);
      renderPalette();
    } else if (e.key === "ArrowUp") {
      e.preventDefault();
      sel = Math.max(sel - 1, 0);
      renderPalette();
    } else if (e.key === "Enter") {
      e.preventDefault();
      runItem(vis[Math.min(sel, vis.length - 1)]);
    } else if (e.key === "Escape") {
      e.preventDefault();
      closePalette();
    }
  });
  for (const it of items) {
    it.addEventListener("click", () => runItem(it));
    it.addEventListener("mousemove", () => {
      const index = visibleItems().indexOf(it);
      if (index >= 0 && index !== sel) {
        sel = index;
        renderPalette();
      }
    });
  }

  window.addEventListener("keydown", (e) => {
    if ((e.metaKey || e.ctrlKey) && (e.key === "k" || e.key === "K")) {
      e.preventDefault();
      if (paletteOpen) closePalette();
      else openPalette();
      return;
    }
    if (paletteOpen) {
      if (e.key === "Escape") {
        e.preventDefault();
        closePalette();
      }
      return;
    }
    const tag = e.target?.tagName ?? "";
    if (/INPUT|TEXTAREA|SELECT/.test(tag) || e.metaKey || e.ctrlKey || e.altKey) return;
    if (e.key === "ArrowRight") step(1);
    else if (e.key === "ArrowLeft") step(-1);
    else if (e.key === "t") toggleMode();
  });

  const tabButtons = [...document.querySelectorAll("[data-tab]")];
  const panes = [...document.querySelectorAll("[data-pane]")];
  for (const b of tabButtons) {
    b.addEventListener("click", () => {
      for (const other of tabButtons) {
        other.setAttribute("aria-selected", String(other === b));
      }
      for (const pane of panes) pane.hidden = pane.dataset.pane !== b.dataset.tab;
    });
  }

  for (const b of portButtons) {
    b.addEventListener("click", () => {
      port = b.dataset.portId;
      for (const other of portButtons) {
        other.setAttribute("aria-pressed", String(other === b));
      }
      updateSteps();
    });
  }

  const canCopy = typeof navigator.clipboard?.writeText === "function";
  for (const btn of document.querySelectorAll("[data-step-copy]")) {
    if (!canCopy) {
      btn.hidden = true;
      continue;
    }
    btn.addEventListener("click", async () => {
      try {
        await navigator.clipboard.writeText(btn.dataset.copyText ?? "");
      } catch {
        return;
      }
      btn.textContent = "copied";
      clearTimeout(copyTimers.get(btn));
      copyTimers.set(btn, setTimeout(() => { btn.textContent = "copy"; }, 1400));
    });
  }

  syncControls();
  rewritePage();
}
