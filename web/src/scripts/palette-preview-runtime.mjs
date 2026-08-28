export function initializeFeaturedPalettePreview() {
  const dataNode = document.querySelector("[data-featured-palette-data]");
  const currentPreview = dataNode?.previousElementSibling;
  if (!(dataNode instanceof HTMLScriptElement)) return;
  if (!(currentPreview instanceof HTMLElement) || !currentPreview.matches("[data-featured-palette-preview]")) return;

  let palettes;
  try {
    palettes = JSON.parse(dataNode.textContent ?? "");
  } catch {
    return;
  }
  const ansiRoles = ["black", "red", "green", "yellow", "blue", "magenta", "cyan", "white"];
  const hasStringProperties = (value, properties) =>
    value !== null && typeof value === "object" && properties.every((property) => typeof value[property] === "string");
  const isPalette = (value) => {
    if (value === null || typeof value !== "object") return false;
    const tokens = value.tokens;
    if (tokens === null || typeof tokens !== "object") return false;
    const ansi = tokens.ansi;
    return (
      hasStringProperties(value, ["id", "name"]) &&
      hasStringProperties(tokens.meta, ["variant"]) &&
      hasStringProperties(tokens.ui, ["background", "surface_dim", "surface", "overlay", "border", "foreground", "muted", "subtle", "accent", "accent_secondary"]) &&
      ansi !== null && typeof ansi === "object" &&
      hasStringProperties(ansi.normal, ansiRoles) && hasStringProperties(ansi.bright, ansiRoles) &&
      hasStringProperties(tokens.semantic, ["green", "yellow", "magenta"])
    );
  };
  if (!Array.isArray(palettes) || palettes.length === 0 || !palettes.every(isPalette)) return;

  const palette = palettes[Math.floor(Math.random() * palettes.length)];
  const nextPreview = currentPreview.cloneNode(true);
  if (!(nextPreview instanceof HTMLElement)) return;
  const compact = nextPreview.dataset.previewLayout === "compact";
  const nextSwatches = [
    ["neutral", "BACKGROUND", "ui.background", palette.tokens.ui.background], ["neutral", "SURFACE DIM", "ui.surface_dim", palette.tokens.ui.surface_dim],
    ["neutral", "SURFACE", "ui.surface", palette.tokens.ui.surface], ["neutral", "OVERLAY", "ui.overlay", palette.tokens.ui.overlay],
    ["neutral", "MUTED", "ui.muted", palette.tokens.ui.muted], ["neutral", "FOREGROUND", "ui.foreground", palette.tokens.ui.foreground],
    ["accent", "ACCENT", "ui.accent", palette.tokens.ui.accent], ["accent", "SECONDARY", "ui.accent_secondary", palette.tokens.ui.accent_secondary],
    ["accent", "YELLOW", "semantic.yellow", palette.tokens.semantic.yellow],
    ...ansiRoles.map((role) => ["normal", role.toUpperCase(), `ansi.normal.${role}`, palette.tokens.ansi.normal[role]]),
    ...ansiRoles.map((role) => ["bright", `BR ${role.toUpperCase()}`, `ansi.bright.${role}`, palette.tokens.ansi.bright[role]]),
  ];
  const code = [
    "const theme = {", `  id: "${palette.id}",`, `  mode: "${palette.tokens.meta.variant}",`, "  ui: {",
    `    background: "${palette.tokens.ui.background}",`, `    foreground: "${palette.tokens.ui.foreground}",`, `    accent: "${palette.tokens.ui.accent}",`, "  },",
    ...(compact ? [] : ["  semantic: {", `    red: "${palette.tokens.semantic.red}",`, `    green: "${palette.tokens.semantic.green}",`, `    cyan: "${palette.tokens.semantic.cyan}",`, "  },"]),
    "} as const;",
  ];
  const terminal = [`$ sf2-themes show ${palette.id}`, `theme       ${palette.id}`, `mode        ${palette.tokens.meta.variant}`, `background  ${palette.tokens.ui.background}`, `foreground  ${palette.tokens.ui.foreground}`, `accent      ${palette.tokens.ui.accent}`, `ansi.white  ${palette.tokens.ansi.normal.white}`];
  const syntaxPattern = /\/\/.*$|"(?:\\.|[^"])*"|\b(?:const|as)\b|\b(?:id|mode|ui|background|foreground|accent|semantic|red|green|cyan)(?=\s*:)|\b\d+(?:\.\d+)?\b|[{}(),:.;]/g;
  const highlightCodeLine = (line) => {
    const tokens = [];
    let cursor = 0;
    for (const match of line.matchAll(syntaxPattern)) {
      const index = match.index ?? 0;
      if (index > cursor) tokens.push(["plain", line.slice(cursor, index)]);
      const text = match[0];
      let kind = "punctuation";
      if (text.startsWith("//")) kind = "comment";
      else if (text.startsWith('"')) kind = "string";
      else if (/^(const|as)$/.test(text)) kind = "keyword";
      else if (/^(id|mode|ui|background|foreground|accent|semantic|red|green|cyan)$/.test(text)) kind = "property";
      else if (/^\d/.test(text)) kind = "number";
      tokens.push([kind, text]);
      cursor = index + text.length;
    }
    if (cursor < line.length) tokens.push(["plain", line.slice(cursor)]);
    return tokens;
  };
  const codeContent = nextPreview.querySelector("[data-code-content]");
  if (codeContent instanceof HTMLElement) {
    codeContent.replaceChildren();
    code.forEach((line, lineIndex) => {
      const lineNode = document.createElement("li");
      const numberNode = document.createElement("span");
      numberNode.setAttribute("aria-hidden", "true");
      numberNode.textContent = String(lineIndex + 1).padStart(2, "0");
      lineNode.append(numberNode);
      const textNode = document.createElement("code");
      highlightCodeLine(line).forEach(([kind, text]) => {
        const tokenNode = document.createElement("span");
        tokenNode.className = `syntax-token syntax-token--${kind}`;
        tokenNode.dataset.syntaxKind = kind;
        tokenNode.dataset.syntaxToken = "";
        tokenNode.textContent = text;
        textNode.append(tokenNode);
      });
      lineNode.append(textNode);
      codeContent.append(lineNode);
    });
  }
  const styleValues = [
    ["--preview-bg", palette.tokens.ui.background], ["--preview-surface", palette.tokens.ui.surface], ["--preview-overlay", palette.tokens.ui.overlay],
    ["--preview-border", palette.tokens.ui.border], ["--preview-fg", palette.tokens.ui.foreground], ["--preview-muted", palette.tokens.ui.muted],
    ["--preview-subtle", palette.tokens.ui.subtle], ["--preview-accent", palette.tokens.ui.accent], ["--preview-secondary", palette.tokens.ui.accent_secondary],
    ["--preview-code-keyword", palette.tokens.semantic.magenta], ["--preview-code-property", palette.tokens.ui.accent], ["--preview-code-string", palette.tokens.semantic.green],
    ["--preview-code-number", palette.tokens.semantic.yellow], ["--preview-code-comment", palette.tokens.ui.subtle], ["--preview-code-punctuation", palette.tokens.ui.muted],
  ];
  nextPreview.dataset.selectedPalette = palette.id;
  nextPreview.dataset.previewId = palette.id;
  nextPreview.setAttribute("aria-label", `Featured palette: ${palette.name}`);
  styleValues.forEach(([property, value]) => nextPreview.style.setProperty(property, value));
  nextPreview.querySelectorAll("[data-preview-id]").forEach((node) => { node.textContent = palette.id; });
  const name = nextPreview.querySelector("[data-preview-name]");
  const mode = nextPreview.querySelector("[data-preview-mode]");
  const terminalContent = nextPreview.querySelector("[data-terminal-content]");
  if (name) name.textContent = palette.name;
  if (mode) mode.textContent = palette.tokens.meta.variant.toUpperCase();
  if (terminalContent) terminalContent.textContent = terminal.join("\n");
  const swatchNodes = nextPreview.querySelectorAll("[data-preview-swatch]");
  if (swatchNodes.length !== nextSwatches.length) return;
  swatchNodes.forEach((node, index) => {
    if (!(node instanceof HTMLElement)) return;
    const [group, label, token, value] = nextSwatches[index];
    node.dataset.swatchGroup = group;
    node.dataset.swatchToken = token;
    node.style.setProperty("--swatch", value);
    node.setAttribute("aria-label", `${label} ${value}`);
    const labelNode = node.querySelector("[data-swatch-label]");
    const valueNode = node.querySelector("[data-swatch-value]");
    if (labelNode) labelNode.textContent = label;
    if (valueNode) valueNode.textContent = value;
  });
  currentPreview.replaceWith(nextPreview);
}
