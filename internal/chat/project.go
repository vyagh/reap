package chat

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// DirnameFor is the folder name Claude Code gives the project at cwd: every
// character that is not an ASCII letter or digit becomes a dash, so a dot or
// an underscore changes too.
func DirnameFor(cwd string) string {
	var b strings.Builder
	for _, r := range cwd {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' {
			b.WriteRune(r)
		} else {
			b.WriteByte('-')
		}
	}
	return b.String()
}

// Pretty is the name the list shows for a project folder: the folder name without
// the home prefix and leading dashes, ~ for the home folder itself. A name that
// would come out empty is shown whole.
func Pretty(h Homes, dir string) string {
	base := lastPart(dir)
	prefix := DirnameFor(h.Home)
	if base == prefix {
		return "~"
	}
	name := strings.TrimLeft(strings.TrimPrefix(base, prefix), "-")
	if name == "" {
		return base
	}
	return name
}

// lastPart is what follows the last separator. Unlike filepath.Base it is empty
// for a path that ends in one, as in 0.5.0.
func lastPart(path string) string {
	return path[strings.LastIndexByte(path, filepath.Separator)+1:]
}

// SplitPath is the leaf and the parent of a chat's working folder, the home folder
// written as ~. A folder with no cwd gives fallback and no parent. The home test
// is plain text, so /home/u2/x under /home/u gives ~2/x.
func SplitPath(h Homes, cwd, fallback string) (leaf, parent string) {
	if cwd == "" {
		return fallback, ""
	}
	p := cwd
	if strings.HasPrefix(cwd, h.Home) {
		p = "~" + cwd[len(h.Home):]
	}
	p = strings.TrimRight(p, string(filepath.Separator))
	leaf = lastPart(p)
	if leaf == "" {
		return p, ""
	}
	return leaf, dirPart(p)
}

// dirPart is everything before the last separator without the separators that end
// it, unless nothing else is left, as in 0.5.0.
func dirPart(path string) string {
	head := path[:strings.LastIndexByte(path, filepath.Separator)+1]
	if strings.Trim(head, string(filepath.Separator)) == "" {
		return head
	}
	return strings.TrimRight(head, string(filepath.Separator))
}

func projectsDir(h Homes) string {
	return filepath.Join(h.Claude, "projects")
}

// UnderProj reports whether dir is a real folder directly inside the projects
// folder once symlinks are resolved, so a link pointing out of the tree is never
// read or deleted through.
func UnderProj(h Homes, dir string) bool {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return false
	}
	real, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return false
	}
	fi, err := os.Stat(real)
	if err != nil || !fi.IsDir() {
		return false
	}
	proj, err := filepath.EvalSymlinks(projectsDir(h))
	return err == nil && filepath.Dir(real) == proj
}

// ProjectDirs is every project folder that passes UnderProj, in name order. Names
// starting with a dot are left out, as 0.5.0 does.
func ProjectDirs(h Homes) []string {
	entries, _ := os.ReadDir(projectsDir(h))
	var dirs []string
	for _, e := range entries {
		d := filepath.Join(projectsDir(h), e.Name())
		if !strings.HasPrefix(e.Name(), ".") && UnderProj(h, d) {
			dirs = append(dirs, d)
		}
	}
	return dirs
}

// dirByCwd finds the project whose first transcript with a cwd recorded this
// cwd. Claude Code caps a long folder name and adds a hash reap cannot
// reproduce, so the name alone cannot be trusted.
func dirByCwd(h Homes, cwd string, cwdOf func(string) string) string {
	for _, d := range ProjectDirs(h) {
		entries, _ := os.ReadDir(d)
		for _, e := range entries {
			if strings.HasPrefix(e.Name(), ".") || !strings.HasSuffix(e.Name(), ".jsonl") {
				continue
			}
			if c := cwdOf(filepath.Join(d, e.Name())); c != "" {
				if c == cwd {
					return d
				}
				break
			}
		}
	}
	return ""
}

// ResolveDir turns what the user typed into a project folder. No argument means
// the project of the current folder, an existing folder means its project, and
// anything else is a piece of a project name where the one that ends with it wins
// a tie. With strict set, a folder that points out of the projects tree is
// refused.
func ResolveDir(h Homes, arg string, strict bool, cwdOf func(transcript string) string) (string, error) {
	proj := projectsDir(h)
	d := ""
	if arg == "" {
		wd, err := PhysicalWd()
		if err != nil {
			return "", err
		}
		d = filepath.Join(proj, DirnameFor(wd))
		if fi, err := os.Stat(d); err != nil || !fi.IsDir() {
			if found := dirByCwd(h, wd, cwdOf); found != "" {
				d = found
			}
		}
	} else if fi, err := os.Stat(arg); err == nil && fi.IsDir() {
		ap := arg
		if !filepath.IsAbs(arg) {
			wd, err := PhysicalWd()
			if err != nil {
				return "", err
			}
			ap = filepath.Join(wd, arg)
		}
		ap = filepath.Clean(ap)
		d = filepath.Join(proj, DirnameFor(ap))
		if filepath.Dir(ap) == proj {
			d = ap
		}
	}
	if d != "" {
		if fi, err := os.Stat(d); err == nil && fi.IsDir() && strict && !UnderProj(h, d) {
			return "", fmt.Errorf("%s points outside %s, refusing", d, proj)
		}
		return d, nil
	}
	var cands []string
	for _, c := range ProjectDirs(h) {
		if strings.Contains(filepath.Base(c), arg) {
			cands = append(cands, c)
		}
	}
	if len(cands) > 1 {
		var exact []string
		for _, c := range cands {
			if strings.HasSuffix(filepath.Base(c), arg) {
				exact = append(exact, c)
			}
		}
		if len(exact) == 1 {
			cands = exact
		}
	}
	switch len(cands) {
	case 1:
		return cands[0], nil
	case 0:
		return "", fmt.Errorf("no project folder matches '%s'", arg)
	}
	names := make([]string, len(cands))
	for i, c := range cands {
		names[i] = filepath.Base(c)
	}
	return "", fmt.Errorf("ambiguous '%s':\n  %s", arg, strings.Join(names, "\n  "))
}

// PhysicalWd is the current folder with symlinks resolved. os.Getwd would
// return $PWD when it names the same folder, and Claude Code names a project
// after the real path.
func PhysicalWd() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	return filepath.EvalSymlinks(wd)
}
