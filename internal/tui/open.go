package tui

import (
	"os"
	"os/exec"
)

func init() {
	keys["Enter"] = (*ui).open
	afterRun = (*ui).startLaunch
}

// open sets u.launch for the chat under the cursor and leaves the screen. The
// refusals come in this order: open elsewhere, folder gone, command not on PATH.
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

func (u *ui) startLaunch() error {
	if u.launch == nil {
		return nil
	}
	return start(*u.launch)
}

// env is the agent's environment, with PWD and OLDPWD set as a shell's cd would.
func (l launch) env() []string {
	old := os.Getenv("PWD")
	if old == "" {
		old, _ = os.Getwd()
	}
	return append(os.Environ(), "OLDPWD="+old, "PWD="+l.dir)
}
