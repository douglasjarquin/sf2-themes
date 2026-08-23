"""Typed errors for the theme pack."""


class ThemeError(Exception):
    """A catalog, validation, filesystem, or CLI error.

    Not a frozen dataclass: CPython assigns `__traceback__` on raise.
    """

    def __init__(self, message: str) -> None:
        super().__init__(message)
        self.message = message


class CliError(ThemeError):
    """An expected command-line failure."""
