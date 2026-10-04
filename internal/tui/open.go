package tui

import (
	"os"
	"os/exec"
)

func init() {
	keys["Enter"] = (*ui).open
	afterRun = (*ui).startLaunch
}

// open checks the chat under the cursor and, when nothing stops it, sets
// u.launch and leaves the screen (reap:1686-1694). The refusals come in
// this order: open elsewhere, folder gone, command not on PATH.
func (u *ui) open() {
	c := u.f.at(u.cur)
	if c == nil {
		return
	}
	argv, dir, err := u.start.Agents.For(c.Source).ResumeCmd(*c)
	switch {
	case err != nil:
		u.msg = message{text: err.Error()}
	case c.Live:
		u.msg = message{text: "already open elsewhere"}
	case dir == "" || !isDir(dir):
		u.msg = message{text: "folder gone: " + orQuestion(dir)}
	default:
		if _, err := exec.LookPath(argv[0]); err != nil {
			u.msg = message{text: argv[0] + " not found on PATH"}
			return
		}
		u.launch = &launch{argv, dir}
		u.quit = true
	}
}

func orQuestion(dir string) string {
	if dir == "" {
		return "?"
	}
	return dir
}

func isDir(path string) bool {
	fi, err := os.Stat(path)
	return err == nil && fi.IsDir()
}

// startLaunch starts the chat Enter opened, once the screen is closed. It
// does nothing when the user left without opening one.
func (u *ui) startLaunch() error {
	if u.launch == nil {
		return nil
	}
	return start(*u.launch)
}

// env is the environment the agent starts with: the folder is the working
// directory, and PWD and OLDPWD say so the way a shell's cd does (reap:632-634).
func (l launch) env() []string {
	old := os.Getenv("PWD")
	if old == "" {
		old, _ = os.Getwd()
	}
	return append(os.Environ(), "OLDPWD="+old, "PWD="+l.dir)
}
