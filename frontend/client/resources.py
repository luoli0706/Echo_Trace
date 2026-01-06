from __future__ import annotations

import sys
from pathlib import Path


def _bundle_root() -> Path:
    # PyInstaller sets sys.frozen and sys._MEIPASS.
    if getattr(sys, "frozen", False) and hasattr(sys, "_MEIPASS"):
        return Path(getattr(sys, "_MEIPASS")).resolve()
    return Path(__file__).resolve().parents[2]  # .../Echo_Trace


def project_root() -> Path:
    # In dev: .../Echo_Trace
    # In bundle: <_MEIPASS>
    return _bundle_root()


def frontend_root() -> Path:
    # In dev: .../Echo_Trace/frontend
    # In bundle: <_MEIPASS>
    if getattr(sys, "frozen", False) and hasattr(sys, "_MEIPASS"):
        return _bundle_root()
    return Path(__file__).resolve().parents[1]


def asset_path(*parts: str) -> str:
    return str(frontend_root().joinpath("assets", *parts))


def config_path(*parts: str) -> str:
    return str(project_root().joinpath("config", *parts))


def legacy_game_config_path() -> str:
    return str(project_root().joinpath("game_config.json"))
