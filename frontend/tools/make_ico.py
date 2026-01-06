from __future__ import annotations

from pathlib import Path


def main() -> int:
    try:
        from PIL import Image
    except Exception as e:
        print(f"Pillow not installed: {e}")
        return 2

    frontend_dir = Path(__file__).resolve().parents[1]
    assets_dir = frontend_dir / "assets"
    src_png = assets_dir / "icos" / "Treasure_Box.png"
    out_ico = assets_dir / "app.ico"

    if not src_png.exists():
        print(f"Source icon not found: {src_png}")
        return 1

    assets_dir.mkdir(parents=True, exist_ok=True)

    img = Image.open(src_png).convert("RGBA")
    sizes = [(16, 16), (24, 24), (32, 32), (48, 48), (64, 64), (128, 128), (256, 256)]

    # Pillow can write multi-size ICO when passing sizes.
    img.save(out_ico, format="ICO", sizes=sizes)
    print(f"Wrote {out_ico}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
