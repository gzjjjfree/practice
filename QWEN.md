必须牢记并严格遵守以下原则：

1. **以暗猜接口为耻，以认真查阅为荣**：不凭函数名猜测，所有接口、参数、返回值必须来自项目源码、类型定义或官方文档。
2. **以模糊执行为耻，以寻求确认为荣**：遇到不明确的需求或边界时，先向用户确认，而不是自行脑补。
3. **以盲想业务为耻，以人类确认为荣**：不擅自决定业务规则（如"不可用"的具体定义），业务逻辑必须由人类确认。
4. **以创造接口为耻，以复用现有为荣**：优先复用项目中已有的代码、组件和工具，避免重复造轮子。
5. **以跳过验证为耻，以主动测试为荣**：代码生成后必须运行单元测试、类型检查或构建命令，验证通过才算完成任务。
6. **以破坏架构为耻，以遵循规范为荣**：严格遵守项目现有的目录结构、命名规范、技术栈和设计模式，不随意跨层调用。
7. **以假装理解为耻，以诚实无知为荣**：遇到无法验证的内容或不确定的知识时，坦白未知，不给出虚假的确切结论。
8. **以盲目修改为耻，以谨慎重构为荣**：只修改完成当前任务所必需的代码，小步修改，不顺手重构无关模块。

---

# Project Context: 题库练习系统 (Practice App)

## Overview

A cross-platform desktop/mobile GUI application built with **Go + Fyne v2.7.x** for local question-bank practice and management. Users import exam/question files (Excel, TXT, JSON) and practice with swipe navigation, wrong-answer tracking, favorites, search, and export.

## Architecture

```
main.go                  — Entry point, AppState struct, showHome(), ClickableBox widget
parser.go               — File parsing engine (XLSX/XLS/TXT/JSON), header detection, data cleaning
common.go               — Shared utilities: hexColor, IsAnswerCorrect, syncQuestionsToState, deleteBank, dialogs
constants.go            — UI dimensions, color palette (light theme), gesture thresholds
correct.go              — Question correction mode: edit form, save/delete operations
export.go               — Export to Excel (.xlsx) and JSON formats
getBox.go               — Custom ClickableBox button component, getTitle() header
save_load.go            — Wrong-set/favorites/practice record persistence (read/write JSON)
showPractice.go         — Main practice UI: question card, options, navigation, stats modal
showSearchPage.go       — Search page with keyword highlighting, type filtering, debounce
showTypeSelection.go    — Mode + type selection intermediate page
swipeable.go            — TouchInterceptor: horizontal swipe → page flip, vertical → scroll

core/                   — Refactored module (constants, helpers) — NOT YET WIRED UP
parser/                 — Refactored module (parser logic) — NOT YET WIRED UP
storage/                — Refactored module (storage logic) — NOT YET WIRED UP
theme/                  — Refactored module (theme logic) — NOT YET WIRED UP
ui/pages/               — Refactored page components — NOT YET WIRED UP
ui/widgets/             — Refactored UI widgets — NOT YET WIRED UP
```

**Key**: All root-level `.go` files are the active implementation (`package main`). The `core/`, `parser/`, `storage/`, `theme/`, and `ui/` directories are a refactored modular structure that is not yet fully wired up — the root files still contain all the logic.

## Building and Running

```bash
go mod tidy
go run .                    # Run
go build -o practice .      # Build binary
go test ./...               # Run tests
```

**Prerequisites**: Go 1.21+, C compiler (cgo required by Fyne), Linux needs `libgl1-mesa-dev libx11-dev`.

## Key Types

- **`Question`**: ID, Type (单选题/多选题/判断题/填空题/问答题), Content, Options[], Answers[], Difficulty
- **`Option`**: Label (A/B/C...), Text
- **`AppState`**: Global state — questions, current list, user answers, submitted flags, storage keys, wrong/favorite sets, practice records
- **`BankData`**: Import/export container — DisplayName, StorageKey, Questions[]

## Data Flow

1. **Import** → `ParseBytesToBank()` → `BankData` → JSON marshal → Fyne sandbox
2. **Practice** → `AppState.CurrentList` holds question subset; `state.Index` tracks position
3. **Persistence** → Wrong-set/favorites as JSON arrays of question IDs; practice records per-mode in nested maps
4. **Auto-save** → `handleUserSelectOption()` triggers on every answer

## UI Patterns

- **`ClickableBox`**: Custom widget wrapping `fyne.Container` with `canvas.Rectangle` bg + `canvas.Text`. Implements `fyne.Tappable`.
- **`TouchInterceptor`**: Drag handling — horizontal accumulates for swipe threshold (40px), vertical delegates to parent scroll.
- **Auto-advance**: On correct answer, goroutine sleeps 350ms (single) or 500ms (multi) then advances index.
- **Search debounce**: 300ms `time.AfterFunc` prevents excessive re-renders.
- **Forced theme**: `forcedDarkTheme` implements `fyne.Theme` to override system colors for light-mode rendering.

## File Naming Convention

- Main data: `[ext]Data_[originalName]_[13-digit-ms-timestamp].json`
- Shadow files: `[name]_错题集.json`, `[name]_收藏集.json`, `[name]_答题记录.json`
- Auto-increment alias on duplicate: `题库名(1).xlsx`, `题库名(2).xlsx`

## Recent Changes

Uncommitted changes exist in most root `.go` files. Check `git diff HEAD` before making new changes to avoid conflicts.
