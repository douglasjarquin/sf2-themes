-- Pull in the wezterm API
local wezterm = require("wezterm")

-- This will hold the configuration.
local config = wezterm.config_builder()

-- Catppuccin is built into WezTerm (https://github.com/catppuccin/wezterm).
-- Mocha/Latte track macOS light/dark; nvim and herdr use the same pairing.
local function scheme_for_appearance(appearance)
  if appearance:find("Dark") then
    return "Catppuccin Mocha"
  else
    return "Catppuccin Latte"
  end
end


config.font = wezterm.font("Monaspace Neon")
config.font_size = 16

config.enable_tab_bar = false

config.window_decorations = "RESIZE"

config.window_background_opacity = 1.0
config.macos_window_background_blur = 10

config.window_close_confirmation = "NeverPrompt"

-- and finally, return the configuration to wezterm
local sf2_current = "/home/fighter/.config/sf2-theme/wezterm-current.lua"
wezterm.add_to_config_reload_watch_list(sf2_current)
local sf2_scheme = dofile(sf2_current)
config.color_scheme = sf2_scheme
local sf2_env = config.set_environment_variables or {}
sf2_env.TERM_THEME = (type(sf2_scheme) == "string" and sf2_scheme:find("-light", 1, true)) and "light" or "dark"
config.set_environment_variables = sf2_env
return config
