package tui

// / starts the filter and every key after that goes to typeFilter until Enter or
// Esc. What it matches is in plan (layout.go) and drawRow (list.go).
func init() {
	keys["/"] = (*ui).startFilter
	filterKey = (*ui).typeFilter
}

func (u *ui) startFilter() {
	u.filtering, u.flt = true, ""
}

// typeFilter takes one key while the filter is typed. Only printable ASCII is
// added: curses hands 0.5.0 the bytes of any other character one by one and none
// is accepted, so "é" is ignored. Every key sends the cursor to the top, the
// ignored ones too.
//
// Esc takes effect at once. tcell reports a lone Esc after waiting 50 ms to see
// whether more bytes follow. 0.5.0 swallows a late terminal reply only outside the
// filter, so there is nothing to swallow here.
func (u *ui) typeFilter(k string) {
	switch {
	case k == "Enter":
		u.filtering = false
	case k == "Esc":
		u.filtering, u.flt = false, ""
	case k == "BSpace":
		u.flt = u.flt[:max(0, len(u.flt)-1)]
	case len(k) == 1 && ' ' <= k[0] && k[0] < 127:
		u.flt += k
	}
	u.cur, u.top = 0, 0
}
