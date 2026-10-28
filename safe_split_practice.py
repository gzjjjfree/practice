import os

with open("ui/pages/home/practiceRelated/showPractice.go", "r", encoding="utf-8") as f:
    content = f.read()

lines = content.split('\n')
clean_lines = []
in_import = False
for line in lines:
    stripped = line.strip()
    if stripped.startswith("package "):
        continue
    if stripped.startswith("import ("):
        in_import = True
        continue
    if in_import:
        if stripped == ")":
            in_import = False
        continue
    if not in_import:
        clean_lines.append(line)

# Find a blank line at brace depth 0 around line 450 to split ShowPractice into two functions or parts
# Or even simpler: keep ShowPractice in showPractice_render.go and ShowQuestionModal in showPractice_interaction.go, 
# and let's check if 884 lines is fine or if we should add a helper function.
# Wait, let's check if any rule forbids a file from being 800 lines if it's a single file. 
# But to be safe and strictly adhere to target ≤600 lines, let's split ShowPractice by extracting an inner helper function or splitting the file cleanly.

print("Safe split script")
