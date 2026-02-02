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

	// Get upstream branch info
	bi, err := fetchBranchInfo(repoConfig.RepoPath)
	if err != nil {
		return tracerr.Wrap(err)
	}

	// Pull with rebase and autostash - explicitly specify remote/branch to avoid
	// "Cannot rebase onto multiple branches" error
	if bi.UpstreamBranch != "" && bi.UpstreamRemote != "" {
		_, err = GitCommand(repoConfig, []string{"pull", "--rebase", "--autostash", bi.UpstreamRemote, bi.UpstreamBranch})
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
