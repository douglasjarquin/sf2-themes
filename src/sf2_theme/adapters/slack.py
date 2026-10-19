"""Slack custom-theme import line. Slack stores the theme in the client, not a file."""

from sf2_theme.model import Theme


def render_theme(theme: Theme) -> str:
    """Return the four-color Import line for one catalog theme."""
    colors = (theme.ui.background, theme.ui.accent, theme.semantic.green, theme.semantic.red)
    return ",".join(color.lower() for color in colors)
