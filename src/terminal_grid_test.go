package fzf

import (
	"math"
	"reflect"
	"testing"

	"github.com/junegunn/fzf/src/util"
)

func gridTestTerminalWithText(grid int, texts []string, spans ...int) *Terminal {
	items := make([]Item, len(spans))
	results := make([]Result, len(spans))
	for idx, span := range spans {
		text := "x"
		if idx < len(texts) {
			text = texts[idx]
		}
		items[idx] = Item{text: util.ToChars([]byte(text)), span: span}
		items[idx].text.Index = int32(idx)
		results[idx] = Result{item: &items[idx]}
	}
	return &Terminal{
		grid:   grid,
		merger: NewMerger(nil, [][]Result{results}, false, false, revision{}, 0, int32(len(results))),
	}
}

func gridTestTerminal(spans ...int) *Terminal {
	return gridTestTerminalWithText(6, nil, spans...)
}

func TestGridLayoutMixedSpans(t *testing.T) {
	term := gridTestTerminal(1, 5, 1, 2, 4, 6, 1)
	layout := term.gridLayout()
	got := make([][4]int, len(layout.cells))
	for idx, cell := range layout.cells {
		got[idx] = [4]int{cell.index, cell.row, cell.col, cell.span}
	}
	want := [][4]int{
		{0, 0, 0, 1}, {1, 0, 1, 5},
		{2, 1, 0, 1}, {3, 1, 1, 2},
		{4, 2, 0, 4}, {5, 3, 0, 6},
		{6, 4, 0, 1},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("placements = %#v, want %#v", got, want)
	}
	if !reflect.DeepEqual(layout.rowStarts, []int{0, 2, 4, 5, 6}) {
		t.Fatalf("row starts = %#v", layout.rowStarts)
	}
}

func TestGridLayoutSpanOneCompatibility(t *testing.T) {
	term := gridTestTerminal(1, 1, 1, 1, 1, 1, 1)
	layout := term.gridLayout()
	for idx, cell := range layout.cells {
		if cell.row != idx/6 || cell.col != idx%6 || cell.span != 1 {
			t.Fatalf("cell %d = %#v", idx, cell)
		}
	}
}

func TestGridWidthsLargeListRetainsFloors(t *testing.T) {
	spans := make([]int, 4097)
	texts := make([]string, 4097)
	for idx := range spans {
		spans[idx] = 1
		texts[idx] = "x"
	}
	term := gridTestTerminalWithText(3, texts, spans...)
	term.pointerLen, term.markerLen, term.gridGap = 1, 1, 2
	widths := term.gridColumnWidthsForWidth(term.gridLayout(), 9)
	if !reflect.DeepEqual(widths, []int{5, 0, 0}) {
		t.Fatalf("large narrow widths=%#v", widths)
	}
}

func TestGridNearestSelectableSkipsBlankScrollbarRow(t *testing.T) {
	term := gridTestTerminalWithText(2, []string{"top", " ", "\t", "bottom"}, 2, 1, 1, 2)
	layout := term.gridLayout()
	if got := term.gridNearestSelectable(layout, 1, 1); got != 3 {
		t.Fatalf("nearest selectable=%d want 3", got)
	}
}

func TestGridWidthsPreserveRenderableFloors(t *testing.T) {
	term := gridTestTerminalWithText(3, []string{"one", "two", "three", "spanning"}, 1, 1, 1, 3)
	term.pointerLen, term.markerLen, term.gridGap = 1, 1, 2
	layout := term.gridLayout()
	widths := term.gridColumnWidthsForWidth(layout, 15)
	for col, width := range widths {
		if width < term.gridCellOverhead()+1 {
			t.Fatalf("track %d width=%d below renderable floor %d", col, width, term.gridCellOverhead()+1)
		}
	}
	if sumInts(widths) > 15 {
		t.Fatalf("widths %#v exceed available width", widths)
	}
}

func TestGridWidthsCollapseWholeTracksWhenFloorInfeasible(t *testing.T) {
	term := gridTestTerminalWithText(3, []string{"one", "two", "three"}, 1, 1, 1)
	term.pointerLen, term.markerLen, term.gridGap = 1, 1, 2
	widths := term.gridColumnWidthsForWidth(term.gridLayout(), 9)
	want := []int{5, 0, 0}
	if !reflect.DeepEqual(widths, want) {
		t.Fatalf("narrow widths=%#v want=%#v", widths, want)
	}
}

func TestGridWidthsAdaptAcrossResize(t *testing.T) {
	term := gridTestTerminalWithText(2, []string{"long value", "other", "wide spanning value"}, 1, 1, 2)
	term.pointerLen, term.markerLen, term.gridGap = 1, 1, 2
	layout := term.gridLayout()
	wide := term.gridColumnWidthsForWidth(layout, 40)
	narrow := term.gridColumnWidthsForWidth(layout, 12)
	if sumInts(wide) <= sumInts(narrow) || sumInts(narrow) > 12 {
		t.Fatalf("resize widths wide=%#v narrow=%#v", wide, narrow)
	}
}

func TestGridVerticalNavigationSkipsBlankRowsAndCycles(t *testing.T) {
	term := gridTestTerminalWithText(2, []string{"top", " ", "\t", "bottom"}, 2, 1, 1, 2)
	term.cy = 0
	if !term.gridVmove(1, false) || term.cy != 3 {
		t.Fatalf("down across blank row selected %d, want 3", term.cy)
	}
	if term.gridVmove(1, false) {
		t.Fatal("non-cycling move past final selectable row succeeded")
	}
	term.cycle = true
	if !term.gridVmove(1, true) || term.cy != 0 {
		t.Fatalf("cycle across blank row selected %d, want 0", term.cy)
	}
}

func TestGridScrollbarUsesLogicalRows(t *testing.T) {
	term := gridTestTerminal(6, 1, 5, 6, 1, 5, 6, 1, 5)
	layout := term.gridLayout()
	if got := gridOffsetForBar(layout, 2, 1, 1); got != layout.rowStarts[len(layout.rows)-2] {
		t.Fatalf("bottom drag offset = %d, want row start %d", got, layout.rowStarts[len(layout.rows)-2])
	}
}

func TestGridScrollbarAlternatingDensity(t *testing.T) {
	term := gridTestTerminal(6, 1, 5, 6, 1, 5, 6, 1, 5, 6)
	layout := term.gridLayout()
	for start := 0; start <= 3; start++ {
		got := gridOffsetForBar(layout, 4, 1, start)
		wantRow := int(math.Ceil(float64(start) * float64(len(layout.rows)-4) / 3.0))
		if got != layout.rowStarts[wantRow] {
			t.Fatalf("bar start %d: offset=%d want=%d", start, got, layout.rowStarts[wantRow])
		}
	}
}

func TestGridMouseHitAcrossEntireSpanAndBlank(t *testing.T) {
	term := gridTestTerminalWithText(6, []string{"left", "details", " "}, 1, 5, 1)
	layout := term.gridLayout()
	widths := []int{4, 3, 3, 3, 3, 3}
	left := widths[0]
	right := sumInts(widths)
	for x := left; x < right; x++ {
		if got := term.gridClickIndexWithLayout(layout, widths, 0, x); got != 1 {
			t.Fatalf("span click x=%d selected %d, want 1", x, got)
		}
	}
	if got := term.gridClickIndexWithLayout(layout, widths, 1, 0); got != -1 {
		t.Fatalf("blank click selected %d", got)
	}
}

func TestGridOffsetRestoresLogicalScreenRowAfterReflow(t *testing.T) {
	before := gridTestTerminal(1, 5, 6, 1, 5, 6)
	beforeLayout := before.gridLayout()
	screenRow := beforeLayout.byIndex[4].row - beforeLayout.byIndex[2].row

	after := gridTestTerminal(6, 1, 5, 1, 5, 6)
	afterLayout := after.gridLayout()
	offset := gridOffsetForScreenRow(afterLayout, 4, screenRow)
	gotRow := afterLayout.byIndex[offset].row
	wantRow := afterLayout.byIndex[4].row - screenRow
	if gotRow != wantRow || offset != afterLayout.rowStarts[wantRow] {
		t.Fatalf("restored offset=%d row=%d want row start %d", offset, gotRow, afterLayout.rowStarts[wantRow])
	}
}

func TestGridItemAtRowAnchorAcrossSpan(t *testing.T) {
	term := gridTestTerminal(1, 5, 2, 2, 2)
	layout := term.gridLayout()
	if got := term.gridItemAtRowAnchor(layout, 1, 5); got != 4 {
		t.Fatalf("right anchor selected %d, want 4", got)
	}
}

func TestGridOffsetFallbacksAfterRemovalAndShrink(t *testing.T) {
	term := gridTestTerminalWithText(6, []string{"a", "b", "c", "d"}, 6, 1, 5, 6)
	layout := term.gridLayout()
	fallback := term.gridNearestSelectable(layout, len(layout.rows)-1, 4)
	if fallback < 0 {
		t.Fatal("no fallback after filter removal")
	}
	offset := gridOffsetForScreenRow(layout, fallback, 1)
	if offset != layout.rowStarts[max(0, layout.byIndex[fallback].row-1)] {
		t.Fatalf("fallback offset=%d", offset)
	}
	last := term.gridNearestSelectable(layout, len(layout.rows)-1, 0)
	if last != 3 {
		t.Fatalf("shrink fallback=%d want 3", last)
	}
}

func TestGridRawModeRestoresLogicalScreenRow(t *testing.T) {
	filtered := gridTestTerminal(1, 5, 6, 1, 5)
	filteredLayout := filtered.gridLayout()
	screenRow := filteredLayout.byIndex[4].row - filteredLayout.byIndex[2].row
	raw := gridTestTerminal(6, 1, 5, 6, 1, 5)
	rawLayout := raw.gridLayout()
	offset := gridOffsetForScreenRow(rawLayout, 4, screenRow)
	if rawLayout.byIndex[offset].row != max(0, rawLayout.byIndex[4].row-screenRow) {
		t.Fatalf("raw-mode offset=%d did not restore screen row", offset)
	}
}

func TestGridToggleMovementSkipsPlaceholderRows(t *testing.T) {
	term := gridTestTerminalWithText(2, []string{"first", " ", "\t", "last"}, 2, 1, 1, 2)
	term.cy = 0
	if !term.gridVmove(1, true) || term.cy != 3 {
		t.Fatalf("toggle geometry selected %d want 3", term.cy)
	}
}

func TestGridJumpTargetsVisibleSelectablePlacements(t *testing.T) {
	term := gridTestTerminalWithText(6, []string{"full", " ", "detail", "next", "last"}, 6, 1, 5, 1, 5)
	layout := term.gridLayout()
	term.offset = layout.rowStarts[1]
	got := term.gridJumpTargets(layout, 2)
	want := []int{2, 3, 4}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("jump targets=%#v want=%#v", got, want)
	}
}

func TestGridOffsetMovementUsesLogicalRows(t *testing.T) {
	term := gridTestTerminal(6, 1, 5, 6, 1, 5)
	layout := term.gridLayout()
	term.offset = layout.rowStarts[1]
	term.cy = 2
	current := layout.byIndex[term.cy]
	offsetRow := layout.byIndex[term.offset].row
	target := term.gridNearestSelectable(layout, offsetRow+1+current.row-offsetRow, current.col)
	if target != 3 {
		t.Fatalf("logical offset target=%d want 3", target)
	}
}

func TestGridOffsetEdgesFallbackToGeometry(t *testing.T) {
	for _, reverse := range []bool{false, true} {
		term := gridTestTerminalWithText(2, []string{"top", " ", "\t", "bottom"}, 2, 1, 1, 2)
		term.layout = layoutDefault
		if reverse {
			term.layout = layoutReverse
		}
		// A zero-height viewport clamps both offsets, forcing cursor fallback.
		term.cy, term.offset = 0, 0
		move := 1
		if reverse {
			move = -1
		}
		term.gridOffsetMove(term.gridLayout(), move, len(term.gridLayout().rows))
		if term.cy != 3 {
			t.Fatalf("reverse=%v top edge selected %d want 3", reverse, term.cy)
		}
		term.cy = 3
		term.gridOffsetMove(term.gridLayout(), -move, len(term.gridLayout().rows))
		if term.cy != 0 {
			t.Fatalf("reverse=%v bottom edge selected %d want 0", reverse, term.cy)
		}
	}
}

func TestGridOffsetMiddleCentersLogicalRows(t *testing.T) {
	term := gridTestTerminalWithText(6,
		[]string{"full", " ", "\t", "middle", "detail", "last"},
		6, 1, 1, 6, 1, 5)
	layout := term.gridLayout()
	// Placeholder-only rows still count as physical viewport rows.
	wantMiddleRow := util.Constrain(layout.byIndex[4].row-1, 0, max(0, len(layout.rows)-3))
	if got := gridOffsetForMiddle(layout, 4, 3); got != layout.rowStarts[wantMiddleRow] {
		t.Fatalf("middle offset=%d want row start %d", got, layout.rowStarts[wantMiddleRow])
	}
	// Clamp at both viewport edges.
	if got := gridOffsetForMiddle(layout, 0, 3); got != layout.rowStarts[0] {
		t.Fatalf("top middle offset=%d", got)
	}
	if got := gridOffsetForMiddle(layout, 5, 3); got != layout.rowStarts[max(0, len(layout.rows)-3)] {
		t.Fatalf("bottom middle offset=%d", got)
	}
}

func TestGridRawModeEmptyLayoutSafe(t *testing.T) {
	term := gridTestTerminalWithText(6, nil)
	layout := term.gridLayout()
	if len(layout.byIndex) != 0 || gridOffsetForScreenRow(layout, 0, 0) != 0 {
		t.Fatalf("empty raw layout=%#v", layout)
	}
}

func TestGridVerticalNavigationUsesOverlap(t *testing.T) {
	term := gridTestTerminal(1, 5, 2, 2, 2)
	term.cy = 1 // row 0, tracks 1..5
	if !term.gridVmove(1, false) || term.cy != 2 {
		t.Fatalf("down from span-5 selected %d, want 2", term.cy)
	}
	term.cy = 4 // row 1, tracks 4..5
	if !term.gridVmove(-1, false) || term.cy != 1 {
		t.Fatalf("up from right cell selected %d, want 1", term.cy)
	}
}
