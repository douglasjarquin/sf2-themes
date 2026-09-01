import {
  normalizeStoredFamily,
  resolveThemeVariant,
  SITE_THEME_STORAGE_KEY,
} from "../data/site-theme.mjs";
import { themeFamilies } from "../data/theme-data.mjs";

function currentScheme(mediaQuery) {
  return mediaQuery.matches ? "light" : "dark";
}

function safeStorage() {
  try {
    return window.localStorage;
  } catch {
    return null;
  }
}

export function initializeSiteThemeControl() {
  const control = document.querySelector("[data-site-theme-control]");
  const select = control?.querySelector("[data-site-theme-select]");
  const mode = control?.querySelector("[data-site-theme-mode]");
  const swatches = control?.querySelectorAll("[data-site-theme-swatch]");
  const status = control?.querySelector("[data-site-theme-status]");
  if (!(control instanceof HTMLElement)
    || !(select instanceof HTMLSelectElement)
    || !(mode instanceof HTMLElement)
    || !(swatches instanceof NodeList)
    || !(status instanceof HTMLElement)
    || control.dataset.siteThemeInitialized === "true") return;
  control.dataset.siteThemeInitialized = "true";

  const mediaQuery = window.matchMedia("(prefers-color-scheme: light)");
  const storage = safeStorage();

  const apply = (familyId, announce) => {
    const scheme = currentScheme(mediaQuery);
    const family = normalizeStoredFamily(familyId, themeFamilies);
    const variant = resolveThemeVariant(family, scheme, themeFamilies);
    document.documentElement.dataset.siteThemeFamily = family;
    document.documentElement.dataset.siteThemeId = variant.id;
    document.documentElement.dataset.siteColorScheme = scheme;
    select.value = family;
    mode.textContent = `FOLLOWS SYSTEM · ${scheme.toUpperCase()}`;
    swatches.forEach((swatch, index) => {
      swatch.style.setProperty("--site-theme-swatch", variant.siteTokens.preview[index]);
    });
    if (announce) status.textContent = `SITE THEME: ${family.toUpperCase()} · FOLLOWS SYSTEM · ${scheme.toUpperCase()}`;
  };

  let storedFamily = null;
  try { storedFamily = storage?.getItem(SITE_THEME_STORAGE_KEY) ?? null; } catch {}
  apply(storedFamily, false);
  select.addEventListener("change", () => {
    const family = normalizeStoredFamily(select.value, themeFamilies);
    try { storage?.setItem(SITE_THEME_STORAGE_KEY, family); } catch {}
    apply(family, true);
  });
  mediaQuery.addEventListener("change", () => apply(select.value, false));
}
