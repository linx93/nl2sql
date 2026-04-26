from __future__ import annotations

import argparse
import os
from functools import lru_cache
from pathlib import Path

from PIL import Image, ImageDraw, ImageFont


ROOT = Path(__file__).resolve().parent.parent
DEFAULT_OUTPUT = ROOT / "docs" / "assets" / "nl2sql-\u5168\u5c40\u67b6\u6784\u56fe.jpg"
REGULAR_FONT_CANDIDATES = (
    Path(r"C:\Windows\Fonts\msyh.ttc"),
    Path(r"C:\Windows\Fonts\simhei.ttf"),
    Path("/System/Library/Fonts/PingFang.ttc"),
    Path("/System/Library/Fonts/STHeiti Light.ttc"),
    Path("/usr/share/fonts/opentype/noto/NotoSansCJK-Regular.ttc"),
    Path("/usr/share/fonts/opentype/noto/NotoSerifCJK-Regular.ttc"),
    Path("/usr/share/fonts/truetype/wqy/wqy-zenhei.ttc"),
    Path("/usr/share/fonts/truetype/arphic/uming.ttc"),
)
BOLD_FONT_CANDIDATES = (
    Path(r"C:\Windows\Fonts\msyhbd.ttc"),
    Path(r"C:\Windows\Fonts\simhei.ttf"),
    Path("/System/Library/Fonts/PingFang.ttc"),
    Path("/System/Library/Fonts/STHeiti Medium.ttc"),
    Path("/usr/share/fonts/opentype/noto/NotoSansCJK-Bold.ttc"),
    Path("/usr/share/fonts/opentype/noto/NotoSerifCJK-Bold.ttc"),
    Path("/usr/share/fonts/truetype/wqy/wqy-zenhei.ttc"),
    Path("/usr/share/fonts/truetype/arphic/uming.ttc"),
)

WIDTH = 2200
HEIGHT = 1400
BACKGROUND = "#F6F8FB"
TEXT = "#123049"
MUTED = "#4C647B"
FLOW_FILL = "#EAF3FF"
FLOW_BORDER = "#2A6DB4"
CONFIG_FILL = "#EAF7EF"
CONFIG_BORDER = "#2D8A57"
TOOL_FILL = "#FFF4E2"
TOOL_BORDER = "#C88622"
AUDIT_FILL = "#F6EAFE"
AUDIT_BORDER = "#8A4FC9"
DATA_FILL = "#FFEAEA"
DATA_BORDER = "#C24848"
CATALOG_FILL = "#EEF6F4"
CATALOG_BORDER = "#2E7F78"
ARROW = "#5D7389"
RULE_FILL = "#FFFDF2"
RULE_BORDER = "#B09A2A"


LABEL = {
    "title": "\u81ea\u7136\u8bed\u8a00\u8f6c\u67e5\u8be2\u670d\u52a1\u5168\u5c40\u67b6\u6784\u56fe",
    "subtitle": "\u8986\u76d6\u5728\u7ebf\u67e5\u8be2\u4e3b\u94fe\u8def\u3001\u914d\u7f6e\u7ef4\u62a4\u94fe\u8def\u3001\u5ba1\u8ba1\u94fe\u8def\u4e0e\u53ea\u8bfb\u6267\u884c\u8fb9\u754c",
    "config_layer": "\u914d\u7f6e\u5c42",
    "flow_layer": "\u5728\u7ebf\u67e5\u8be2\u4e3b\u94fe\u8def",
    "tool_layer": "\u914d\u7f6e\u7ef4\u62a4\u4e0e\u5185\u90e8\u5de5\u5177",
    "audit_layer": "\u5ba1\u8ba1\u3001\u5b89\u5168\u4e0e\u6570\u636e\u6e90\u8fb9\u754c",
    "data_source_config": "\u6570\u636e\u6e90\u914d\u7f6e",
    "schema_snapshot": "\u6570\u636e\u5e93\u7ed3\u6784\u5feb\u7167",
    "semantic_config": "\u9886\u57df\u8bed\u4e49\u914d\u7f6e",
    "runtime_catalog": "\u8fd0\u884c\u65f6\u8bed\u4e49\u76ee\u5f55",
    "catalog_note": "\u7edf\u4e00\u88c5\u8f7d\u6307\u6807\u3001\u7ef4\u5ea6\u3001\u660e\u7ec6\u89c6\u56fe\u3001\u89d2\u8272\u4e0e\u522b\u540d",
    "user_request": "\u7528\u6237\u8bf7\u6c42",
    "api_service": "\u63a5\u53e3\u670d\u52a1",
    "orchestrator": "\u7f16\u6392\u670d\u52a1",
    "planner": "\u89c4\u5212\u5668",
    "planner_sub": "\u751f\u6210\u539f\u59cb\u8ba1\u5212",
    "resolver": "\u89e3\u6790\u5668",
    "resolver_sub": "\u751f\u6210\u89c4\u8303\u8ba1\u5212",
    "builder": "\u67e5\u8be2\u8bed\u53e5\u6784\u5efa\u5668",
    "guard": "\u5b88\u536b\u6821\u9a8c",
    "executor": "\u53ea\u8bfb\u6267\u884c\u5668",
    "formatter": "\u7ed3\u679c\u683c\u5f0f\u5316",
    "response": "\u63a5\u53e3\u54cd\u5e94",
    "catalog_input": "\u76ee\u5f55\u8f93\u5165",
    "semantic_load": "\u8bed\u4e49\u4e0e\u6743\u9650\u88c5\u8f7d",
    "tool_cli": "\u5185\u90e8\u547d\u4ee4\u884c\u5de5\u5177",
    "tool_test": "\u6570\u636e\u6e90\u6d4b\u8bd5",
    "tool_pull": "\u7ed3\u6784\u5feb\u7167\u62c9\u53d6",
    "tool_scaffold": "\u8bed\u4e49\u811a\u624b\u67b6\u751f\u6210",
    "tool_validate": "\u914d\u7f6e\u6821\u9a8c",
    "tool_note": "\u7528\u4e8e\u7ef4\u62a4\u53ea\u8bfb\u6570\u636e\u6e90\u3001\u751f\u6210\u6570\u636e\u5e93\u7ed3\u6784\u5feb\u7167\u3001\n\u4fdd\u5b88\u751f\u6210\u8bed\u4e49\u914d\u7f6e\u5e76\u505a\u4e00\u81f4\u6027\u6821\u9a8c",
    "audit_log": "\u5ba1\u8ba1\u65e5\u5fd7",
    "audit_note": "\u8bb0\u5f55\u81ea\u7136\u8bed\u8a00\u95ee\u9898\u3001\u539f\u59cb\u8ba1\u5212\u3001\u89c4\u8303\u8ba1\u5212\u3001\n\u6784\u5efa\u540e\u7684\u67e5\u8be2\u8bed\u53e5\u3001\u6821\u9a8c\u540e\u7684\u67e5\u8be2\u8bed\u53e5\u3001\u7ed3\u679c\u6458\u8981\u4e0e\u9519\u8bef\u4fe1\u606f",
    "database_source": "\u53ea\u8bfb\u6570\u636e\u5e93\u6570\u636e\u6e90",
    "database_note": "\u5355\u6b21\u8bf7\u6c42\u53ea\u547d\u4e2d\u4e00\u4e2a\u6570\u636e\u6e90",
    "execute_query": "\u6267\u884c\u67e5\u8be2",
    "rules_title": "\u786c\u7ea6\u675f",
    "rules_body": "\u2022 \u6a21\u578b\u53ea\u751f\u6210\u539f\u59cb\u8ba1\u5212\uff0c\u4e0d\u76f4\u63a5\u751f\u6210\u67e5\u8be2\u8bed\u53e5\n\u2022 \u6240\u6709\u67e5\u8be2\u8bed\u53e5\u5fc5\u987b\u7ecf\u8fc7\u5b88\u536b\u6821\u9a8c\n\u2022 \u67e5\u8be2\u6267\u884c\u53ea\u4f7f\u7528\u53ea\u8bfb\u6570\u636e\u6e90",
}


def font(size: int, bold: bool = False) -> ImageFont.FreeTypeFont:
    path = resolve_font_path("bold" if bold else "regular")
    return ImageFont.truetype(str(path), size=size)


@lru_cache(maxsize=None)
def resolve_font_path(weight: str) -> Path:
    env_var = "NL2SQL_ARCH_FONT_BOLD" if weight == "bold" else "NL2SQL_ARCH_FONT_REGULAR"
    override = os.environ.get(env_var)
    if override:
        path = Path(override)
        if path.exists():
            return path
        raise FileNotFoundError(f"font override does not exist: {path}")

    candidates = BOLD_FONT_CANDIDATES if weight == "bold" else REGULAR_FONT_CANDIDATES
    for candidate in candidates:
        if candidate.exists():
            return candidate

    raise FileNotFoundError(
        "no usable Chinese font found; set NL2SQL_ARCH_FONT_REGULAR and NL2SQL_ARCH_FONT_BOLD to existing font files"
    )


def text_center(
    draw: ImageDraw.ImageDraw,
    box: tuple[int, int, int, int],
    content: str,
    size: int,
    fill: str = TEXT,
    bold: bool = False,
) -> None:
    box_w = box[2] - box[0]
    box_h = box[3] - box[1]
    fnt = font(size, bold=bold)
    lines = content.split("\n")
    line_heights: list[int] = []
    max_width = 0
    for line in lines:
        bbox = draw.textbbox((0, 0), line, font=fnt)
        max_width = max(max_width, bbox[2] - bbox[0])
        line_heights.append(bbox[3] - bbox[1])
    total_height = sum(line_heights) + (len(lines) - 1) * 8
    y = box[1] + (box_h - total_height) / 2
    for idx, line in enumerate(lines):
        bbox = draw.textbbox((0, 0), line, font=fnt)
        line_width = bbox[2] - bbox[0]
        draw.text((box[0] + (box_w - line_width) / 2, y), line, font=fnt, fill=fill)
        y += line_heights[idx] + 8


def text_top_left(
    draw: ImageDraw.ImageDraw,
    position: tuple[int, int],
    content: str,
    size: int,
    fill: str = TEXT,
    bold: bool = False,
    line_gap: int = 8,
) -> None:
    fnt = font(size, bold=bold)
    x, y = position
    for line in content.split("\n"):
        draw.text((x, y), line, font=fnt, fill=fill)
        bbox = draw.textbbox((x, y), line, font=fnt)
        y = bbox[3] + line_gap


def rounded_box(
    draw: ImageDraw.ImageDraw,
    box: tuple[int, int, int, int],
    fill: str,
    outline: str,
    radius: int = 24,
    width: int = 4,
) -> None:
    draw.rounded_rectangle(box, radius=radius, fill=fill, outline=outline, width=width)


def arrow(
    draw: ImageDraw.ImageDraw,
    start: tuple[int, int],
    end: tuple[int, int],
    color: str = ARROW,
    width: int = 5,
    head: int = 14,
) -> None:
    draw.line([start, end], fill=color, width=width)
    if start[0] == end[0]:
        direction = 1 if end[1] > start[1] else -1
        draw.polygon(
            [
                (end[0], end[1]),
                (end[0] - head, end[1] - direction * head),
                (end[0] + head, end[1] - direction * head),
            ],
            fill=color,
        )
    else:
        direction = 1 if end[0] > start[0] else -1
        draw.polygon(
            [
                (end[0], end[1]),
                (end[0] - direction * head, end[1] - head),
                (end[0] - direction * head, end[1] + head),
            ],
            fill=color,
        )


def elbow_arrow(
    draw: ImageDraw.ImageDraw,
    start: tuple[int, int],
    turn: tuple[int, int],
    end: tuple[int, int],
    color: str = ARROW,
    width: int = 5,
) -> None:
    draw.line([start, turn, end], fill=color, width=width)
    arrow(draw, turn, end, color=color, width=width)


def draw_module(
    draw: ImageDraw.ImageDraw,
    box: tuple[int, int, int, int],
    title: str,
    fill: str,
    outline: str,
    subtitle: str | None = None,
) -> None:
    rounded_box(draw, box, fill=fill, outline=outline)
    text_center(draw, box, title if subtitle is None else f"{title}\n{subtitle}", 26, bold=True)


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Generate the NL2SQL architecture diagram.")
    parser.add_argument("--output", type=Path, default=DEFAULT_OUTPUT, help="Output JPG path.")
    parser.add_argument("--print-fonts", action="store_true", help="Print resolved regular and bold font paths, then exit.")
    return parser.parse_args()


def main() -> None:
    args = parse_args()

    if args.print_fonts:
        print(resolve_font_path("regular"))
        print(resolve_font_path("bold"))
        return

    output = args.output
    output.parent.mkdir(parents=True, exist_ok=True)
    image = Image.new("RGB", (WIDTH, HEIGHT), BACKGROUND)
    draw = ImageDraw.Draw(image)

    text_top_left(draw, (70, 36), LABEL["title"], 44, bold=True)
    text_top_left(draw, (72, 94), LABEL["subtitle"], 24, fill=MUTED)

    rounded_box(draw, (45, 150, 2155, 360), "#F0F8F4", "#D3EADA", radius=30, width=2)
    text_top_left(draw, (65, 166), LABEL["config_layer"], 28, bold=True, fill=CONFIG_BORDER)

    rounded_box(draw, (45, 390, 2155, 835), "#F9FBFE", "#DCE7F2", radius=30, width=2)
    text_top_left(draw, (65, 406), LABEL["flow_layer"], 28, bold=True, fill=FLOW_BORDER)

    rounded_box(draw, (45, 865, 1320, 1335), "#FFFAF2", "#F2DFC4", radius=30, width=2)
    text_top_left(draw, (65, 882), LABEL["tool_layer"], 28, bold=True, fill=TOOL_BORDER)

    rounded_box(draw, (1355, 865, 2155, 1335), "#FCF7FF", "#E3D5F7", radius=30, width=2)
    text_top_left(draw, (1375, 882), LABEL["audit_layer"], 28, bold=True, fill=AUDIT_BORDER)

    config_boxes = [
        ((95, 220, 365, 305), LABEL["data_source_config"]),
        ((405, 220, 675, 305), LABEL["schema_snapshot"]),
        ((715, 220, 985, 305), LABEL["semantic_config"]),
        ((1115, 205, 1465, 320), LABEL["runtime_catalog"]),
    ]
    for box, title in config_boxes:
        fill = CONFIG_FILL if title != LABEL["runtime_catalog"] else CATALOG_FILL
        outline = CONFIG_BORDER if title != LABEL["runtime_catalog"] else CATALOG_BORDER
        draw_module(draw, box, title, fill, outline)

    text_top_left(draw, (1128, 338), LABEL["catalog_note"], 20, fill=MUTED)
    arrow(draw, (365, 262), (405, 262))
    arrow(draw, (675, 262), (715, 262))
    arrow(draw, (985, 262), (1115, 262))

    flow_y = 585
    flow_boxes = [
        ((80, 515, 250, 650), LABEL["user_request"]),
        ((295, 515, 465, 650), LABEL["api_service"]),
        ((510, 515, 700, 650), LABEL["orchestrator"]),
        ((745, 500, 965, 665), LABEL["planner"], LABEL["planner_sub"]),
        ((1010, 500, 1230, 665), LABEL["resolver"], LABEL["resolver_sub"]),
        ((1275, 500, 1475, 665), LABEL["builder"]),
        ((1520, 500, 1710, 665), LABEL["guard"]),
        ((1755, 500, 1965, 665), LABEL["executor"]),
        ((2010, 500, 2125, 665), LABEL["formatter"]),
    ]
    for item in flow_boxes:
        if len(item) == 2:
            box, title = item
            draw_module(draw, box, title, FLOW_FILL, FLOW_BORDER)
        else:
            box, title, subtitle = item
            draw_module(draw, box, title, FLOW_FILL, FLOW_BORDER, subtitle)

    response_box = (1840, 710, 2115, 790)
    draw_module(draw, response_box, LABEL["response"], FLOW_FILL, FLOW_BORDER)

    for idx in range(len(flow_boxes) - 1):
        left = flow_boxes[idx][0]
        right = flow_boxes[idx + 1][0]
        arrow(draw, (left[2], flow_y), (right[0], flow_y))
    elbow_arrow(draw, (2125, flow_y), (2140, flow_y), (2140, 750))
    arrow(draw, (2140, 750), (2115, 750))

    arrow(draw, (1290, 320), (1290, 500))
    arrow(draw, (1200, 320), (620, 500))
    text_top_left(draw, (1210, 428), LABEL["catalog_input"], 18, fill=MUTED)
    text_top_left(draw, (1010, 390), LABEL["semantic_load"], 18, fill=MUTED)

    tool_boxes = [
        ((90, 960, 280, 1060), LABEL["tool_cli"]),
        ((360, 920, 585, 1010), LABEL["tool_test"]),
        ((360, 1035, 585, 1125), LABEL["tool_pull"]),
        ((620, 920, 845, 1010), LABEL["tool_scaffold"]),
        ((620, 1035, 845, 1125), LABEL["tool_validate"]),
    ]
    for box, title in tool_boxes:
        draw_module(draw, box, title, TOOL_FILL, TOOL_BORDER)

    arrow(draw, (280, 1010), (360, 965))
    arrow(draw, (280, 1010), (360, 1080))
    arrow(draw, (585, 965), (620, 965))
    arrow(draw, (585, 1080), (620, 1080))

    text_top_left(draw, (104, 1085), LABEL["tool_note"], 21, fill=MUTED)
    elbow_arrow(draw, (470, 920), (470, 845), (220, 360))
    elbow_arrow(draw, (470, 1125), (470, 845), (530, 360))
    elbow_arrow(draw, (730, 920), (730, 845), (830, 360))
    elbow_arrow(draw, (730, 1125), (730, 845), (1180, 360))

    audit_box = (1405, 950, 1715, 1085)
    draw_module(draw, audit_box, LABEL["audit_log"], AUDIT_FILL, AUDIT_BORDER)
    text_top_left(draw, (1408, 1098), LABEL["audit_note"], 21, fill=MUTED)

    data_box = (1760, 950, 2095, 1085)
    draw_module(draw, data_box, LABEL["database_source"], DATA_FILL, DATA_BORDER)
    text_top_left(draw, (1785, 1098), LABEL["database_note"], 21, fill=MUTED)

    arrow(draw, (605, 650), (605, 950))
    arrow(draw, (1965, 585), (1760, 585))
    elbow_arrow(draw, (1860, 665), (1860, 790), (1860, 950))
    text_top_left(draw, (1820, 815), LABEL["execute_query"], 18, fill=MUTED)

    rules_box = (1405, 1175, 2095, 1295)
    rounded_box(draw, rules_box, RULE_FILL, RULE_BORDER, radius=26, width=3)
    text_top_left(
        draw,
        (1430, 1200),
        f"{LABEL['rules_title']}\n{LABEL['rules_body']}",
        24,
        fill="#6F5E0E",
        line_gap=10,
    )

    image.save(output, format="JPEG", quality=92)
    print(f"saved: {output}")


if __name__ == "__main__":
    main()
