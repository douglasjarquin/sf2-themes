from pathlib import Path

from sf2_theme.adapters.nvim import apply_nvim, render_pointer, render_scheme, setup_nvim
from sf2_theme.catalog import get_theme, parse_catalog
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
    assert {path.stem.removeprefix("street-fighter-ii-") for path in colors.glob("*.lua")} == {
        theme.metadata.id for theme in catalog
    }
    assert "main" in (tmp_path / "nvim" / "sf2-theme" / "current.lua").read_text(encoding="utf-8")
    assert len(results) == len(catalog) + 1


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
    assert "colorscheme street-fighter-ii-main" in (
        tmp_path / "nvim" / "sf2-theme" / "current.lua"
    ).read_text(encoding="utf-8")
    assert len(results) == len(catalog) + 2


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

    assert capsys.readouterr().out.splitlines()[-1] == "ryu-light"
    backups = list((config_dir / "sf2-theme").glob("current.lua.bak.*"))
    assert len(backups) == 1
    assert "main" in backups[0].read_text(encoding="utf-8")


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

    assert dispatch(
        [
            "apply",
            "nvim",
            "--theme",
            "ryu-light",
            "--config-dir",
            str(config_dir),
            "--dry-run",
        ]
    ) == 0

    assert pointer.read_text(encoding="utf-8") == original
    assert not list((config_dir / "sf2-theme").glob("current.lua.bak.*"))


def test_apply_refuses_symlink_without_follow_flag(tmp_path: Path, capsys) -> None:
    config_dir = tmp_path / "nvim"
    colors = config_dir / "colors"
    colors.mkdir(parents=True)
    target = tmp_path / "target.lua"
    target.write_text("-- preserve\n", encoding="utf-8")
    (colors / "street-fighter-ii-main.lua").symlink_to(target)

    assert dispatch(["apply", "nvim", "--config-dir", str(config_dir)]) == 1

    assert "symlink" in capsys.readouterr().err
    assert target.read_text(encoding="utf-8") == "-- preserve\n"


def test_apply_follows_symlink_when_requested(tmp_path: Path) -> None:
    config_dir = tmp_path / "nvim"
    colors = config_dir / "colors"
    colors.mkdir(parents=True)
    target = tmp_path / "target.lua"
    target.write_text("-- preserve\n", encoding="utf-8")
    (colors / "street-fighter-ii-main.lua").symlink_to(target)

    assert dispatch(
        ["apply", "nvim", "--config-dir", str(config_dir), "--follow-symlinks"]
    ) == 0

    assert (colors / "street-fighter-ii-main.lua").is_symlink()
    assert 'vim.g.colors_name = "street-fighter-ii-main"' in target.read_text(encoding="utf-8")


def test_config_dir_and_environment_override(tmp_path: Path, monkeypatch) -> None:
    env_dir = tmp_path / "env-nvim"
    monkeypatch.setenv("NVIM_CONFIG_DIR", str(env_dir))

    assert dispatch(["apply", "nvim"]) == 0

    assert (env_dir / "sf2-theme" / "current.lua").is_file()
    assert not (tmp_path / "nvim" / "sf2-theme").exists()


def test_pointer_rendering_is_a_colorscheme_switch() -> None:
    theme = get_theme("main", parse_catalog())

    assert "colorscheme street-fighter-ii-main" in render_pointer(theme)
