package commands

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/user/helm-env/internal/config"
)

// DoctorHelp prints help for the doctor command.
func DoctorHelp() {
	fmt.Println(`Usage: helm-env doctor

Run a set of diagnostic checks against the current helm-env installation.
Each check prints one line prefixed with [OK], [WARN], or [FAIL].  The
command exits 0 if no FAIL is reported, 1 otherwise.`)
}

// httpClient is the interface used by Doctor's network checks.  It is a
// package-level variable so tests can substitute a stub.
type httpClient interface {
	Head(url string) (*http.Response, error)
}

var doctorHTTPClient httpClient = &http.Client{Timeout: 3 * time.Second}

// Doctor runs diagnostic checks and prints a report.
func Doctor() error {
	return doctorWithWriter(os.Stdout)
}

func doctorWithWriter(w io.Writer) error {
	fails := 0

	report := func(level, desc, detail string) {
		if detail == "" {
			fmt.Fprintf(w, "[%s] %s\n", level, desc)
		} else {
			fmt.Fprintf(w, "[%s] %s: %s\n", level, desc, detail)
		}
		if level == "FAIL" {
			fails++
		}
	}

	// 1. HELMENV_ROOT set.
	root := os.Getenv("HELMENV_ROOT")
	if root == "" {
		report("FAIL", "HELMENV_ROOT is set", "unset")
	} else {
		report("OK", "HELMENV_ROOT is set", root)
	}

	// 2. $HELMENV_ROOT exists and is writable.
	if root != "" {
		info, err := os.Stat(root)
		switch {
		case err != nil:
			report("FAIL", "$HELMENV_ROOT exists", err.Error())
		case !info.IsDir():
			report("FAIL", "$HELMENV_ROOT exists", "not a directory")
		default:
			if writable(root) {
				report("OK", "$HELMENV_ROOT is writable", "")
			} else {
				report("FAIL", "$HELMENV_ROOT is writable", "not writable")
			}
		}
	}

	// 3. versions directory.
	if root != "" {
		versionsDir := filepath.Join(root, "versions")
		if info, err := os.Stat(versionsDir); err != nil || !info.IsDir() {
			report("FAIL", "$HELMENV_ROOT/versions exists", "missing")
		} else {
			report("OK", "$HELMENV_ROOT/versions exists", "")
		}
	}

	// 4. shim exists and is executable.
	if root != "" {
		shim := filepath.Join(root, "shims", "helm")
		info, err := os.Stat(shim)
		switch {
		case err != nil:
			report("FAIL", "$HELMENV_ROOT/shims/helm exists", err.Error())
		case info.Mode()&0o111 == 0:
			report("FAIL", "$HELMENV_ROOT/shims/helm is executable", "not executable")
		default:
			report("OK", "$HELMENV_ROOT/shims/helm exists and is executable", "")
		}
	}

	// 5. shims dir is on PATH BEFORE any other helm.
	if root != "" {
		shimDir := filepath.Join(root, "shims")
		pathEnv := os.Getenv("PATH")
		found := false
		for _, p := range splitPath(pathEnv) {
			if p == shimDir {
				found = true
				break
			}
		}
		if !found {
			report("FAIL", "$HELMENV_ROOT/shims is on $PATH", "not on PATH")
		} else {
			// Check that `helm` on PATH is the shim.
			resolved, err := exec.LookPath("helm")
			if err != nil {
				report("WARN", "helm is found on $PATH", err.Error())
			} else {
				if resolved == filepath.Join(shimDir, "helm") {
					report("OK", "$HELMENV_ROOT/shims precedes other helm on $PATH", "")
				} else {
					report("FAIL", "$HELMENV_ROOT/shims precedes other helm on $PATH",
						"another helm found first at "+resolved)
				}
			}
		}
	}

	// 6. Cache parseable.
	if root != "" {
		cacheFile := filepath.Join(root, "cache", "releases.json")
		if _, err := os.Stat(cacheFile); err == nil {
			data, readErr := os.ReadFile(cacheFile)
			if readErr != nil {
				report("WARN", "cache file is readable", readErr.Error())
			} else {
				var payload any
				if jerr := json.Unmarshal(data, &payload); jerr != nil {
					report("FAIL", "cache file is parseable", jerr.Error())
				} else {
					report("OK", "cache file is parseable", "")
				}
			}
		} else {
			report("OK", "cache file (optional)", "not present")
		}
	}

	// 7. Reachability (WARN on failure).
	for _, host := range []string{"https://github.com", "https://get.helm.sh"} {
		if err := checkReachable(host); err != nil {
			report("WARN", "reachable: "+host, err.Error())
		} else {
			report("OK", "reachable: "+host, "")
		}
	}

	// 8. At least one version installed.
	if root != "" {
		installed, err := config.InstalledVersions()
		if err != nil || len(installed) == 0 {
			report("WARN", "at least one helm version installed", "none installed")
		} else {
			report("OK", fmt.Sprintf("installed helm versions (%d)", len(installed)), "")
		}
	}

	// 9. A resolvable version exists.
	if root != "" {
		if v, err := config.ResolveConcreteVersion(); err != nil {
			var nie *config.NotInstalledError
			if errors.As(err, &nie) {
				report("WARN", "active version resolves", err.Error())
			} else {
				report("WARN", "active version resolves", err.Error())
			}
		} else {
			report("OK", "active version resolves", v)
		}
	}

	if fails > 0 {
		return fmt.Errorf("%d check(s) failed", fails)
	}
	return nil
}

// writable reports whether dir is writable by the current user.
func writable(dir string) bool {
	f, err := os.CreateTemp(dir, ".helmenv-doctor-")
	if err != nil {
		return false
	}
	name := f.Name()
	_ = f.Close()
	_ = os.Remove(name)
	return true
}

// splitPath splits a PATH-like env variable using the OS separator.
func splitPath(p string) []string {
	sep := string(os.PathListSeparator)
	if p == "" {
		return nil
	}
	return strings.Split(p, sep)
}

// checkReachable performs a short HEAD request.
func checkReachable(url string) error {
	resp, err := doctorHTTPClient.Head(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	// Any 2xx or 3xx response means the host is reachable.
	if resp.StatusCode >= 400 && resp.StatusCode < 500 {
		// 4xx for a HEAD on a bare host is still "reachable" semantically.
	}
	return nil
}
