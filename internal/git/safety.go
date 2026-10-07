package git

import (
	"os"
	"path/filepath"
	"runtime"
)

func safetyArgs() []string {
	hooks := emptyHooksPath()
	return []string{
		"-c", "core.hooksPath=" + hooks,
		"-c", "protocol.file.allow=never",
		"-c", "credential.helper=",
		"-c", "core.fsmonitor=false",
		"-c", "advice.detachedHead=false",
	}
}

func emptyHooksPath() string {
	// An empty hooks path is the polite way of telling Git that the repository
	// does not get to run programs on our behalf.
	if runtime.GOOS == "windows" {
		return filepath.Join(os.TempDir(), "forkledger-empty-hooks")
	}
	return os.DevNull
}

func safeEnv() []string {
	home := os.Getenv("HOME")
	if home == "" {
		home = os.Getenv("USERPROFILE")
	}
	env := []string{
		"GIT_TERMINAL_PROMPT=0",
		"GIT_ASKPASS=",
		"GIT_CONFIG_NOSYSTEM=1",
		"LC_ALL=C",
		"LANG=C",
		"PATH=" + os.Getenv("PATH"),
		"SystemRoot=" + os.Getenv("SystemRoot"),
		"HOME=" + home,
	}
	if runtime.GOOS == "windows" {
		env = append(env,
			"USERPROFILE="+os.Getenv("USERPROFILE"),
			"HOMEDRIVE="+os.Getenv("HOMEDRIVE"),
			"HOMEPATH="+os.Getenv("HOMEPATH"),
			"TMP="+os.Getenv("TMP"),
			"TEMP="+os.Getenv("TEMP"),
			"ComSpec="+os.Getenv("ComSpec"),
			"PATHEXT="+os.Getenv("PATHEXT"),
		)
	}
	if dir, err := os.UserCacheDir(); err == nil {
		env = append(env, "XDG_CACHE_HOME="+dir)
	}
	return env
}

func EnsureEmptyHooks() error {
	if runtime.GOOS != "windows" {
		return nil
	}
	return os.MkdirAll(emptyHooksPath(), 0o755)
}
