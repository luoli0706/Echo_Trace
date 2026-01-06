# -*- mode: python ; coding: utf-8 -*-

from pathlib import Path

block_cipher = None

frontend_dir = Path(SPECPATH).resolve()  # provided by PyInstaller when executing the spec
project_dir = frontend_dir.parent

assets_dir = frontend_dir / "assets"
config_dir = project_dir / "config"
legacy_cfg = project_dir / "game_config.json"

_datas = []
if assets_dir.exists():
    _datas.append((str(assets_dir), "assets"))
if config_dir.exists():
    _datas.append((str(config_dir), "config"))
if legacy_cfg.exists():
    _datas.append((str(legacy_cfg), "game_config.json"))


a = Analysis(
    [str(frontend_dir / "main.py")],
    pathex=[str(frontend_dir)],
    binaries=[],
    datas=_datas,
    hiddenimports=[],
    hookspath=[],
    hooksconfig={},
    runtime_hooks=[],
    excludes=[],
    win_no_prefer_redirects=False,
    win_private_assemblies=False,
    cipher=block_cipher,
    noarchive=False,
)

pyz = PYZ(a.pure, a.zipped_data, cipher=block_cipher)

exe = EXE(
    pyz,
    a.scripts,
    a.binaries,
    a.zipfiles,
    a.datas,
    [],
    name="Echo_Trace_Client",
    debug=False,
    bootloader_ignore_signals=False,
    strip=False,
    upx=True,
    upx_exclude=[],
    runtime_tmpdir=None,
    console=False,
    disable_windowed_traceback=False,
    argv_emulation=False,
    target_arch=None,
    codesign_identity=None,
    entitlements_file=None,
    icon=str(frontend_dir / "assets" / "app.ico") if (frontend_dir / "assets" / "app.ico").exists() else None,
)
