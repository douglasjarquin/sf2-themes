from pathlib import Path

from sf2_theme.adapters.nvim import apply_nvim, render_pointer, render_scheme, setup_nvim
from sf2_theme.catalog import get_theme, parse_catalog, theme_pair
from sf2_theme.cli import dispatch


def test_render_scheme_uses_catalog_palette() -> None:
    theme = get_theme("ryu-light", parse_catalog())

    rendered = render_scheme(theme)

    assert "vim.api.nvim_set_hl" in rendered
    assert f'background = "{theme.ui.background}"' in rendered
    assert f'foreground = "{theme.ui.foreground}"' in rendered
    assert f'normal_red = "{theme.ansi_normal.red}"' in rendered
    assert f'bright_blue = "{theme.ansi_bright.blue}"' in rendered
    assert 'vim.o.background = "light"' in rendered


def test_apply_writes_every_catalog_colorscheme_and_current_pointer(tmp_path: Path) -> None:
    catalog = parse_catalog()
    results = apply_nvim(
        get_theme("main", catalog),
        catalog,
        config_dir=tmp_path / "nvim",
        dry_run=False,
        follow_symlinks=False,
    )

    colors = tmp_path / "nvim" / "colors"
    assert {path.stem for path in colors.glob("*.lua")} == {f"sf2-{theme.metadata.id}" for theme in catalog}
    assert "sf2-main" in (tmp_path / "nvim" / "sf2-theme" / "current.lua").read_text(encoding="utf-8")
    assert len(results) == len(catalog) + 1


def test_nvim_generated_ids_are_prefixed_and_old_files_are_replaced(tmp_path: Path) -> None:
    catalog = parse_catalog()
    config_dir = tmp_path / "nvim"
    colors = config_dir / "colors"
    colors.mkdir(parents=True)
    (colors / "street-fighter-ii-main.lua").write_text("old managed colorscheme", encoding="utf-8")

    apply_nvim(
        get_theme("main", catalog),
        catalog,
        config_dir=config_dir,
        dry_run=False,
        follow_symlinks=False,
    )

    written_ids = {path.stem for path in colors.glob("*.lua")}
    expected_ids = {f"sf2-{theme.metadata.id}" for theme in catalog}
    assert written_ids == expected_ids
    assert all(theme_id.startswith("sf2-") for theme_id in written_ids)
    assert "-- sf2-themes: sf2-main" in (config_dir / "sf2-theme" / "current.lua").read_text(encoding="utf-8")
    assert not (colors / "street-fighter-ii-main.lua").exists()


def test_setup_writes_managed_loader_and_default_current(tmp_path: Path) -> None:
    catalog = parse_catalog()

    results = setup_nvim(
        get_theme("main", catalog),
        catalog,
        config_dir=tmp_path / "nvim",
        dry_run=False,
        follow_symlinks=False,
    )

    loader = tmp_path / "nvim" / "plugin" / "sf2-theme.lua"
    assert "sf2-theme/current.lua" in loader.read_text(encoding="utf-8")
    assert "colorscheme sf2-main" in (tmp_path / "nvim" / "sf2-theme" / "current.lua").read_text(encoding="utf-8")
    assert "colorscheme sf2-main-light" in (tmp_path / "nvim" / "sf2-theme" / "current.lua").read_text(encoding="utf-8")
    assert "TERM_THEME" in (tmp_path / "nvim" / "sf2-theme" / "current.lua").read_text(encoding="utf-8")
    assert len(results) == len(catalog) + 2


def test_setup_nvim_migrates_existing_unprefixed_pointer(tmp_path: Path) -> None:
    catalog = parse_catalog()
    config_dir = tmp_path / "nvim"
    pointer = config_dir / "sf2-theme" / "current.lua"
    pointer.parent.mkdir(parents=True)
    pointer.write_text('-- sf2-themes: ken\nvim.cmd("colorscheme street-fighter-ii-ken")\n', encoding="utf-8")

    setup_nvim(
        get_theme("main", catalog),
        catalog,
        config_dir=config_dir,
        dry_run=False,
        follow_symlinks=False,
        replace_pointer=False,
    )

    assert "-- sf2-themes: sf2-ken" in pointer.read_text(encoding="utf-8")
    assert "colorscheme sf2-ken" in pointer.read_text(encoding="utf-8")
    assert "colorscheme sf2-ken-light" in pointer.read_text(encoding="utf-8")
    assert "TERM_THEME" in pointer.read_text(encoding="utf-8")


def test_apply_backs_up_current_pointer_and_current_reads_theme(tmp_path: Path, capsys) -> None:
    config_dir = tmp_path / "nvim"
    catalog = parse_catalog()
    setup_nvim(
        get_theme("main", catalog),
        catalog,
        config_dir=config_dir,
        dry_run=False,
        follow_symlinks=False,
    )

    assert dispatch(["apply", "nvim", "--theme", "ryu-light", "--config-dir", str(config_dir)]) == 0
    assert dispatch(["current", "nvim", "--config-dir", str(config_dir)]) == 0

    assert capsys.readouterr().out.splitlines()[-1] == "sf2-ryu"
    backups = list((config_dir / "sf2-theme").glob("current.lua.bak.*"))
    assert len(backups) == 1
    assert "sf2-main" in backups[0].read_text(encoding="utf-8")


def test_apply_dry_run_leaves_existing_pointer_unchanged(tmp_path: Path) -> None:
    config_dir = tmp_path / "nvim"
    catalog = parse_catalog()
    setup_nvim(
        get_theme("main", catalog),
        catalog,
        config_dir=config_dir,
        dry_run=False,
        follow_symlinks=False,
    )
    pointer = config_dir / "sf2-theme" / "current.lua"
    original = pointer.read_text(encoding="utf-8")

    assert (
        dispatch(
            [
                "apply",
                "nvim",
                "--theme",
                "ryu-light",
                "--config-dir",
                str(config_dir),
                "--dry-run",
            ]
        )
        == 0
    )

    assert pointer.read_text(encoding="utf-8") == original
    assert not list((config_dir / "sf2-theme").glob("current.lua.bak.*"))


def test_apply_refuses_symlink_without_follow_flag(tmp_path: Path, capsys) -> None:
    config_dir = tmp_path / "nvim"
    colors = config_dir / "colors"
    colors.mkdir(parents=True)
    target = tmp_path / "target.lua"
    target.write_text("-- preserve\n", encoding="utf-8")
    (colors / "sf2-main.lua").symlink_to(target)

    assert dispatch(["apply", "nvim", "--config-dir", str(config_dir)]) == 1

    assert "symlink" in capsys.readouterr().err
    assert target.read_text(encoding="utf-8") == "-- preserve\n"


def test_apply_follows_symlink_when_requested(tmp_path: Path) -> None:
    config_dir = tmp_path / "nvim"
    colors = config_dir / "colors"
    colors.mkdir(parents=True)
    target = tmp_path / "target.lua"
    target.write_text("-- preserve\n", encoding="utf-8")
    (colors / "sf2-main.lua").symlink_to(target)

    assert dispatch(["apply", "nvim", "--config-dir", str(config_dir), "--follow-symlinks"]) == 0

    assert (colors / "sf2-main.lua").is_symlink()
    assert 'vim.g.colors_name = "sf2-main"' in target.read_text(encoding="utf-8")


def test_config_dir_and_environment_override(tmp_path: Path, monkeypatch) -> None:
    env_dir = tmp_path / "env-nvim"
    monkeypatch.setenv("NVIM_CONFIG_DIR", str(env_dir))

    assert dispatch(["apply", "nvim"]) == 0

    assert (env_dir / "sf2-theme" / "current.lua").is_file()
    assert not (tmp_path / "nvim" / "sf2-theme").exists()


def test_pointer_rendering_auto_switches_the_character_pair() -> None:
    catalog = parse_catalog()
    dark, light = theme_pair(get_theme("main", catalog), catalog)
    pointer = render_pointer(dark, light)
    assert "-- sf2-themes: sf2-main" in pointer
    assert "colorscheme sf2-main" in pointer
    assert "colorscheme sf2-main-light" in pointer
    assert "TERM_THEME" in pointer
