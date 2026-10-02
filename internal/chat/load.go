package chat

import (
	"sort"
	"time"
)

// Load is every agent's chats in the order the list shows them, with Hidden
// set from keep. With opts.Dir set it is the chats of that one project and
// they run newest first. Without it, projects run by their newest chat and
// chats within a project newest first (reap:518-539). Each agent applies
// opts.Tmp, opts.Subagents and opts.Full itself.
func Load(agents Agents, keep map[string]float64, opts ListOpts) ([]Chat, error) {
	var chats []Chat
	for _, a := range agents {
		got, err := a.List(opts)
		if err != nil {
			return nil, err
		}
		for _, c := range got {
			if opts.Dir != "" && c.Dir != opts.Dir {
				continue
			}
			_, c.Hidden = keep[c.ID]
			chats = append(chats, c)
		}
	}
	if opts.Dir != "" {
		sort.SliceStable(chats, func(i, j int) bool { return chats[i].Mod.After(chats[j].Mod) })
		return chats, nil
	}
	recent := make(map[string]time.Time)
	for _, c := range chats {
		if c.Mod.After(recent[c.Proj]) {
			recent[c.Proj] = c.Mod
		}
	}
	sort.SliceStable(chats, func(i, j int) bool {
		a, b := chats[i], chats[j]
		if ra, rb := recent[a.Proj], recent[b.Proj]; !ra.Equal(rb) {
			return ra.After(rb)
		}
		if a.Proj != b.Proj {
			return a.Proj < b.Proj
		}
		return a.Mod.After(b.Mod)
	})
	return chats, nil
}
