# Lazygit adapter E2E transcript

## setup (36 fragments + merge into existing user config with inline comment + double-quoted "*")
User config before setup:
gui: # my ui settings / scrollHeight: 4 / authorColors: "*": #123456 / os: editPreset: nvim

$ sf2-themes setup lazygit --config-dir $D --adopt  -> created 36 themes/sf2-*.yml + updated config.yml
$ sf2-themes apply lazygit --theme vega --config-dir $D -> updated config.yml
$ sf2-themes current lazygit --config-dir $D
sf2-vega

## PyYAML semantic check
OK: gui.theme has all 12 keys, bold active border, single gui.authorColors wildcard, user config preserved
36 fragments parse, all gui.theme(12 keys)+gui.authorColors, 35 distinct activeBorderColor hues

## Real lazygit consumer check (lazygit 0.x, /opt/homebrew/bin/lazygit)
valid merged config -> passes config load, fails only later on: *fs.PathError open /dev/tty: device not configured
control broken config (duplicate wildcard) -> "couldn't be parsed ... mapping key \"*\" already defined"
