import os

def parse_blocks(file_path):
    with open(file_path, 'r', encoding='utf-8') as f:
        lines = f.readlines()

    blocks = []
    current_block = []
    current_comment = []
    brace_depth = 0
    in_block = False

    for line in lines:
        stripped = line.strip()
        if brace_depth == 0:
            if stripped.startswith("//") or stripped.startswith("/*") or stripped == "":
                if in_block:
                    current_block.append(line)
                else:
                    current_comment.append(line)
                continue
            
            if stripped.startswith("func ") or stripped.startswith("type ") or stripped.startswith("var ") or stripped.startswith("const "):
                if current_block:
                    blocks.append("".join(current_block))
                    current_block = []
                in_block = True
                current_block.extend(current_comment)
                current_comment = []
        
        current_block.append(line)
        brace_depth += line.count("{") - line.count("}")
        if brace_depth == 0 and in_block:
            blocks.append("".join(current_block))
            current_block = []
            in_block = False

    if current_block:
        blocks.append("".join(current_block))
    return blocks

def split_file_evenly(src_file, num_parts, pkg_name):
    blocks = parse_blocks(src_file)
    print(f"Parsed {len(blocks)} blocks from {src_file}")

    # distribute blocks into num_parts aiming for balanced line counts
    parts = [[] for _ in range(num_parts)]
    part_lines = [0] * num_parts

    for block in blocks:
        b_lines = block.count('\n')
        # find part with min lines
        min_idx = part_lines.index(min(part_lines))
        parts[min_idx].append(block)
        part_lines[min_idx] += b_lines

    base_dir, filename = os.path.split(src_file)
    stem, _ = os.path.splitext(filename)

    imports = """import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"

	"github.com/gzjjjfree/practice/core"
	"github.com/gzjjjfree/practice/customElements"
	"github.com/gzjjjfree/practice/network"
	"github.com/gzjjjfree/practice/parser"
	"github.com/gzjjjfree/practice/storageRelated"
)

"""

    for i, blk_list in enumerate(parts):
        dest = os.path.join(base_dir, f"{stem}_part{i+1}.go")
        with open(dest, 'w', encoding='utf-8') as f:
            f.write(f"package {pkg_name}\n\n")
            f.write(imports)
            f.write("\n".join(blk_list))
        print(f"Wrote {dest} with {len(blk_list)} blocks, lines: {part_lines[i]}")

    os.remove(src_file)

if __name__ == "__main__":
    split_file_evenly("ui/pages/home/managementRelated/adminExamCreate.go", 6, "managementRelated")
    split_file_evenly("ui/pages/home/managementRelated/bankManage.go", 3, "managementRelated")
    split_file_evenly("ui/pages/home/managementRelated/myExams.go", 3, "managementRelated")
    split_file_evenly("ui/pages/home/practiceRelated/showPractice.go", 2, "practiceRelated")
    print("Even split completed.")
