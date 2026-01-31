package common

import (
	"github.com/gen2brain/beeep"
	"github.com/ztrue/tracerr"
)

func AutoSync(repoConfig RepoConfig) error {
	var err error
	err = ensureGitAuthor(repoConfig)
	if err != nil {
		return tracerr.Wrap(err)
	}

	// Pull with rebase and autostash - handles:
	// 1. Fetching remote changes
	// 2. Stashing any uncommitted local changes
	// 3. Rebasing any local commits onto remote
	// 4. Restoring stashed changes
	_, err = GitCommand(repoConfig, []string{"pull", "--rebase", "--autostash"})
	if err != nil {
		// Check if it's a conflict
		repoPath := repoConfig.RepoPath
		rebasing, _ := isRebasing(repoPath)
		if rebasing {
			GitCommand(repoConfig, []string{"rebase", "--abort"})
			beeep.Alert("Git Auto Sync - Conflict", "Could not rebase for - "+repoPath, "")
			return errRebaseFailed
		}
		return tracerr.Wrap(err)
	}

	// Commit any new local changes
	err = commit(repoConfig)
	if err != nil {
		return tracerr.Wrap(err)
	}

	// Push
	err = push(repoConfig)
	if err != nil {
		return tracerr.Wrap(err)
	}

	return nil
}
