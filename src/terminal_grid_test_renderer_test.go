package fzf

import "github.com/junegunn/fzf/src/tui"

type gridSpanTestRenderer struct {
	chars chan tui.Event
}

func newGridSpanTestRenderer() *gridSpanTestRenderer {
	return &gridSpanTestRenderer{chars: make(chan tui.Event)}
}

func (*gridSpanTestRenderer) DefaultTheme() *tui.ColorTheme { return tui.Default16 }
func (*gridSpanTestRenderer) Init() error                   { return nil }
func (*gridSpanTestRenderer) Resize(func(int) int)          {}
func (*gridSpanTestRenderer) Pause(bool)                    {}
func (*gridSpanTestRenderer) Resume(bool, bool)             {}
func (*gridSpanTestRenderer) Clear()                        {}
func (*gridSpanTestRenderer) RefreshWindows([]tui.Window)   {}
func (*gridSpanTestRenderer) Refresh()                      {}
func (*gridSpanTestRenderer) Close()                        {}
func (*gridSpanTestRenderer) PassThrough(string)            {}
func (*gridSpanTestRenderer) NeedScrollbarRedraw() bool     { return false }
func (*gridSpanTestRenderer) ShouldEmitResizeEvent() bool   { return false }
func (*gridSpanTestRenderer) Bell()                         {}
func (*gridSpanTestRenderer) HideCursor()                   {}
func (*gridSpanTestRenderer) ShowCursor()                   {}
func (r *gridSpanTestRenderer) GetChar(bool) tui.Event      { return <-r.chars }
func (r *gridSpanTestRenderer) CancelGetChar() {
	select {
	case r.chars <- tui.Event{}:
	default:
	}
}
func (*gridSpanTestRenderer) Top() int           { return 0 }
func (*gridSpanTestRenderer) MaxX() int          { return 80 }
func (*gridSpanTestRenderer) MaxY() int          { return 24 }
func (*gridSpanTestRenderer) Size() tui.TermSize { return tui.TermSize{Columns: 80, Lines: 24} }
func (*gridSpanTestRenderer) NewWindow(top, left, width, height int, _ tui.WindowType, _ tui.BorderStyle, _ bool) tui.Window {
	return &gridSpanTestWindow{top: top, left: left, width: width, height: height}
}

type gridSpanTestWindow struct {
	top, left, width, height int
	x, y                     int
}

func (w *gridSpanTestWindow) Top() int                                                  { return w.top }
func (w *gridSpanTestWindow) Left() int                                                 { return w.left }
func (w *gridSpanTestWindow) Width() int                                                { return w.width }
func (w *gridSpanTestWindow) Height() int                                               { return w.height }
func (*gridSpanTestWindow) DrawBorder()                                                 {}
func (*gridSpanTestWindow) DrawHBorder()                                                {}
func (*gridSpanTestWindow) DrawHSeparator(int, tui.WindowType, bool)                    {}
func (*gridSpanTestWindow) PaintSectionFrame(int, int, tui.WindowType, tui.SectionEdge) {}
func (*gridSpanTestWindow) Refresh()                                                    {}
func (*gridSpanTestWindow) FinishFill()                                                 {}
func (w *gridSpanTestWindow) X() int                                                    { return w.x }
func (w *gridSpanTestWindow) Y() int                                                    { return w.y }
func (w *gridSpanTestWindow) EncloseX(x int) bool                                       { return x >= w.left && x < w.left+w.width }
func (w *gridSpanTestWindow) EncloseY(y int) bool                                       { return y >= w.top && y < w.top+w.height }
func (w *gridSpanTestWindow) Enclose(y, x int) bool                                     { return w.EncloseX(x) && w.EncloseY(y) }
func (w *gridSpanTestWindow) Move(y, x int)                                             { w.y, w.x = y, x }
func (w *gridSpanTestWindow) MoveAndClear(y, x int)                                     { w.Move(y, x) }
func (*gridSpanTestWindow) Print(string)                                                {}
func (*gridSpanTestWindow) CPrint(tui.ColorPair, string)                                {}
func (*gridSpanTestWindow) Fill(string) tui.FillReturn                                  { return 0 }
func (*gridSpanTestWindow) CFill(tui.Color, tui.Color, tui.Color, tui.Attr, string) tui.FillReturn {
	return 0
}
func (*gridSpanTestWindow) LinkBegin(string, string) {}
func (*gridSpanTestWindow) LinkEnd()                 {}
func (*gridSpanTestWindow) Erase()                   {}
func (*gridSpanTestWindow) EraseMaybe() bool         { return false }
func (*gridSpanTestWindow) SetWrapSign(string, int)  {}
