// Package reviewhandoff owns the Markdown review-file contract used by the
// review gate.
//
// The package is intentionally a leaf package: it knows how to resolve safe
// file paths, render and parse the handoff file, validate a saved file against
// live gate state, derive the existing approval/fix decision, and render a
// final audit view. Pipeline, TUI, IPC, and AXI packages should pass structured
// state into this package instead of duplicating Markdown or hash logic.
package reviewhandoff
