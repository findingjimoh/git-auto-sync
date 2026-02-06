package common

import (
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/gen2brain/beeep"
	"github.com/ztrue/tracerr"
)

// lockPath returns the shared lock file path for a repo, matching the format
// used by git-auto-pull.sh: ~/.cache/git-sync-{repo with / replaced by -}.lock
func lockPath(repoPath string) string {
	cacheDir := filepath.Join(os.Getenv("HOME"), ".cache")
	sanitized := strings.ReplaceAll(repoPath, "/", "-")
	return filepath.Join(cacheDir, "git-sync-"+sanitized+".lock")
}

func AutoSync(repoConfig RepoConfig) error {
	// Shared lock file with git-auto-pull.sh to prevent concurrent git operations
	lock := lockPath(repoConfig.RepoPath)
	os.MkdirAll(filepath.Dir(lock), 0755)
	f, err := os.OpenFile(lock, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
	if err != nil {
		// Lock held by pull script (or stale) — skip this cycle
		return nil
	}
	f.Close()
	defer os.Remove(lock)

	err = ensureGitAuthor(repoConfig)
	if err != nil {
		return tracerr.Wrap(err)
	}

	// 1. Commit local changes first (no stash needed)
	err = commit(repoConfig)
	if err != nil {
		return tracerr.Wrap(err)
	}

	// 2. Pull with rebase — our commit gets rebased on top of remote
	bi, err := fetchBranchInfo(repoConfig.RepoPath)
	if err != nil {
		return tracerr.Wrap(err)
	}

	if bi.UpstreamBranch != "" && bi.UpstreamRemote != "" {
		_, err = GitCommand(repoConfig, []string{"fetch", bi.UpstreamRemote})
		if err != nil {
			return tracerr.Wrap(err)
		}

		_, err = GitCommand(repoConfig, []string{"rebase", bi.UpstreamRemote + "/" + bi.UpstreamBranch})
		if err != nil {
			repoPath := repoConfig.RepoPath
			rebasing, _ := isRebasing(repoPath)
			if rebasing {
				GitCommand(repoConfig, []string{"rebase", "--abort"})
				beeep.Alert("Git Auto Sync - Conflict", "Could not rebase for - "+repoPath, "")
				log.Println("Rebase conflict in", repoPath, "- aborted, local commit preserved")
				return errRebaseFailed
			}
			return tracerr.Wrap(err)
		}
	}

	// 3. Push
	err = push(repoConfig)
	if err != nil {
		return tracerr.Wrap(err)
	}

	return nil
}
