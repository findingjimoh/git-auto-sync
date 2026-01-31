package common

import (
	"errors"

	"github.com/gen2brain/beeep"
	"github.com/ztrue/tracerr"
)

// Fixed order: fetch → rebase → commit → push
// This ensures commits are always based on the latest remote state,
// preventing ref mismatch errors when multiple devices sync.
func AutoSync(repoConfig RepoConfig) error {
	var err error
	err = ensureGitAuthor(repoConfig)
	if err != nil {
		return tracerr.Wrap(err)
	}

	// 1. Fetch latest from remote first
	err = fetch(repoConfig)
	if err != nil {
		return tracerr.Wrap(err)
	}

	// 2. Rebase local changes onto latest remote
	err = rebase(repoConfig)
	if err != nil {
		if errors.Is(err, errRebaseFailed) {
			repoPath := repoConfig.RepoPath
			err := beeep.Alert("Git Auto Sync - Conflict", "Could not rebase for - "+repoPath, "assets/warning.png")
			if err != nil {
				return tracerr.Wrap(err)
			}
		}
		return tracerr.Wrap(err)
	}

	// 3. Commit new changes (now based on latest remote state)
	err = commit(repoConfig)
	if err != nil {
		return tracerr.Wrap(err)
	}

	// 4. Push
	err = push(repoConfig)
	if err != nil {
		return tracerr.Wrap(err)
	}

	return nil
}
