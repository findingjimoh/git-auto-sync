package common

import (
	"bytes"
	"log"
	"os"
	"os/exec"
	"sort"
	"strings"

	"github.com/ztrue/tracerr"
)

func commit(repoConfig RepoConfig) error {
	repoPath := repoConfig.RepoPath

	// Use shell git for change detection instead of go-git.
	// go-git v4 doesn't support .gitattributes filters (e.g. git-crypt),
	// which causes false positives and missed changes.
	statusOut, err := GitCommand(repoConfig, []string{"status", "--porcelain"})
	if err != nil {
		return tracerr.Wrap(err)
	}

	lines := strings.Split(strings.TrimRight(statusOut.String(), "\n"), "\n")

	hasChanges := false
	commitMsg := []string{}
	for _, line := range lines {
		if len(line) < 4 {
			continue
		}
		// porcelain format: XY <path>  (X=staging, Y=worktree)
		statusCode := line[:2]
		filePath := line[3:]

		// Handle renamed files: "R  old -> new"
		if idx := strings.Index(filePath, " -> "); idx >= 0 {
			filePath = filePath[idx+4:]
		}

		ignore, err := ShouldIgnoreFile(repoPath, filePath)
		if err != nil {
			return tracerr.Wrap(err)
		}

		if ignore {
			continue
		}

		_, err = GitCommand(repoConfig, []string{"add", "--", filePath})
		if err != nil {
			log.Printf("git add skipped: %s (%v)", filePath, err)
			continue
		}
		hasChanges = true

		commitMsg = append(commitMsg, statusCode+" "+filePath)
	}

	sort.Strings(commitMsg)
	msg := strings.Join(commitMsg, "\n")

	if !hasChanges {
		return nil
	}

	_, err = GitCommand(repoConfig, []string{"commit", "-m", msg})
	if err != nil {
		return tracerr.Wrap(err)
	}

	return nil
}

func GitCommand(repoConfig RepoConfig, args []string) (bytes.Buffer, error) {
	repoPath := repoConfig.RepoPath

	var outb, errb bytes.Buffer

	cmd := "git"
	if repoConfig.GitExec != "" {
		cmd = repoConfig.GitExec
	}

	statusCmd := exec.Command(cmd, args...)
	statusCmd.Dir = repoPath
	statusCmd.Stdout = &outb
	statusCmd.Stderr = &errb
	statusCmd.Env = toEnvString(repoConfig)
	err := statusCmd.Run()

	if err != nil {
		fullCmd := "git " + strings.Join(args, " ")
		err := tracerr.Errorf("%w: Command: %s\nStdOut: %s\nStdErr: %s", err, fullCmd, outb.String(), errb.String())
		return outb, err
	}
	return outb, nil
}

func toEnvString(repoConfig RepoConfig) []string {
	if len(repoConfig.Env) == 0 {
		return os.Environ()
	}
	preserve := map[string]bool{"HOME": true, "PATH": true, "SSH_AUTH_SOCK": true}
	vals := append([]string{}, repoConfig.Env...)
	for _, s := range os.Environ() {
		parts := strings.SplitN(s, "=", 2)
		if preserve[parts[0]] {
			vals = append(vals, s)
		}
	}
	return vals
}

func hasEnvVariable(all []string, name string) bool {
	for _, s := range all {
		parts := strings.Split(s, "=")
		k := parts[0]
		if k == name {
			return true
		}
	}
	return false
}
