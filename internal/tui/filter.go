package tui

// The filter (reap:1500-1506, 1605). / starts it and every key after that
// goes to typeFilter until Enter or Esc. What the filter matches and how the
// list narrows is in plan (layout.go) and drawRow (list.go).
func init() {
	keys["/"] = (*ui).startFilter
	filterKey = (*ui).typeFilter
}

// startFilter begins a new filter. It works in the trash view too.
func (u *ui) startFilter() {
	u.filtering, u.flt = true, ""
}

// typeFilter is one key while the filter is being typed. Enter keeps the
// filter and Esc drops it. Only printable ASCII is added: curses hands Python
// the bytes of any other character one by one and none of them is accepted,
// so text like "é" is ignored. Every key sends the cursor back to the top,
// the ignored ones too.
//
// Esc takes effect at once. tcell reports a lone Esc after waiting 50 ms to
// see whether more bytes follow; Python's swallowing of a late terminal reply
// applies only outside the filter, so there is nothing to swallow here.
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
