package main

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	git "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/sandrolain/eventkit/pkg/common"
	toolutil "github.com/sandrolain/eventkit/pkg/toolutil"
	"github.com/spf13/cobra"
)

func sendCommand() *cobra.Command {
	var (
		remote        string
		branch        string
		interval      string
		filename      string
		payload       string
		mime          string
		commitMessage string
		username      string
		password      string
	)

	cmd := &cobra.Command{
		Use:   "send",
		Short: "Periodically commit and push to a git repo",
		RunE: func(cmd *cobra.Command, args []string) error {
			if remote == "" {
				return fmt.Errorf("--remote is required")
			}
			if _, err := time.ParseDuration(interval); err != nil {
				return fmt.Errorf("invalid interval: %w", err)
			}
			return runGitSend(remote, branch, interval, filename, payload, mime, commitMessage, username, password)
		},
	}

	cmd.Flags().StringVar(&remote, "remote", "", "Remote git repository URL (required)")
	cmd.Flags().StringVar(&branch, "branch", "main", "Branch to commit to")
	cmd.Flags().StringVar(&interval, "interval", "10s", "Interval between commits (e.g. 10s, 1m)")
	cmd.Flags().StringVar(&filename, "filename", "data.txt", "File to update in the repo")
	toolutil.AddPayloadFlags(cmd, &payload, "Automated update at {nowtime}", &mime, toolutil.CTText)
	cmd.Flags().StringVar(&commitMessage, "message", "Automated commit", "Commit message")
	cmd.Flags().StringVar(&username, "username", "", "Username for remote repository (optional)")
	cmd.Flags().StringVar(&password, "password", "", "Password or token for remote repository (optional)")

	return cmd
}

func runGitSend(remote, branch, interval, filename, payload, mime, message, username, password string) error {
	ctx, cancel := common.SetupGracefulShutdown()
	defer cancel()

	tmpDir, err := os.MkdirTemp("", "gittool-*")
	if err != nil {
		return fmt.Errorf("create temp dir: %w", err)
	}
	defer func() {
		if err := os.RemoveAll(tmpDir); err != nil {
			slog.Error("Failed to remove temp dir", "error", err)
		}
	}()

	repo, err := cloneOrInitRepo(tmpDir, remote, branch, username, password)
	if err != nil {
		return err
	}

	logger := toolutil.Logger()
	logger.Info("Git tool ready", "remote", remote, "branch", branch, "file", filename, "interval", interval)

	return common.StartPeriodicTask(ctx, interval, func() error {
		if err := doCommit(repo, tmpDir, branch, filename, payload, mime, message, username, password, remote); err != nil {
			logger.Error("Commit error", "error", err)
			return err
		}
		logger.Info("Committed and pushed", "remote", remote, "branch", branch)
		return nil
	})
}

func cloneOrInitRepo(tmpDir, remote, branch, username, password string) (*git.Repository, error) {
	logger := toolutil.Logger()
	logger.Info("Cloning repository", "remote", remote, "branch", branch, "dir", tmpDir)

	cloneOpts := &git.CloneOptions{
		URL:           remote,
		Progress:      os.Stdout,
		SingleBranch:  true,
		ReferenceName: plumbing.NewBranchReferenceName(branch),
	}
	if username != "" && password != "" {
		cloneOpts.Auth = &http.BasicAuth{Username: username, Password: password}
	}

	repo, err := git.PlainClone(tmpDir, false, cloneOpts)
	if err == nil {
		return repo, nil
	}

	if err == git.ErrRepositoryNotExists || err.Error() == "remote repository is empty" || err.Error() == "repository is empty" {
		logger.Info("Remote repository is empty, initializing new repository")
		repo, initErr := git.PlainInit(tmpDir, false)
		if initErr != nil {
			return nil, fmt.Errorf("init repo: %w", initErr)
		}
		_, remoteErr := repo.CreateRemote(&config.RemoteConfig{Name: "origin", URLs: []string{remote}})
		if remoteErr != nil {
			return nil, fmt.Errorf("add remote: %w", remoteErr)
		}
		if err := checkoutOrCreateBranch(repo, branch); err != nil {
			return nil, err
		}
		return repo, nil
	}

	if err.Error() == "couldn't find remote ref \"refs/heads/"+branch+"\"" {
		logger.Info("Remote branch not found, cloning default branch and creating it locally", "branch", branch)
		cloneOpts2 := &git.CloneOptions{URL: remote, Progress: os.Stdout}
		if username != "" && password != "" {
			cloneOpts2.Auth = &http.BasicAuth{Username: username, Password: password}
		}
		repo, err = git.PlainClone(tmpDir, false, cloneOpts2)
		if err != nil {
			return nil, fmt.Errorf("git clone (default): %w", err)
		}
		if err := checkoutOrCreateBranch(repo, branch); err != nil {
			return nil, err
		}
		return repo, nil
	}

	return nil, fmt.Errorf("git clone error: %w", err)
}

func checkoutOrCreateBranch(repo *git.Repository, branch string) error {
	wt, err := repo.Worktree()
	if err != nil {
		return fmt.Errorf("get worktree: %w", err)
	}
	branchRef := plumbing.NewBranchReferenceName(branch)
	err = wt.Checkout(&git.CheckoutOptions{Branch: branchRef, Create: true, Force: true})
	if err != nil {
		return fmt.Errorf("checkout branch '%s': %w", branch, err)
	}
	return nil
}

func doCommit(repo *git.Repository, repoPath, branch, filename, payload, mime, message, username, password, remote string) error {
	filePath := filepath.Join(repoPath, filename)

	content, _, err := toolutil.BuildPayload(payload, mime)
	if err != nil {
		return fmt.Errorf("build payload: %w", err)
	}

	f, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600) // #nosec G304 -- test tool with controlled path
	if err != nil {
		return fmt.Errorf("open file: %w", err)
	}
	defer func() {
		if err := f.Close(); err != nil {
			slog.Error("Failed to close file", "error", err)
		}
	}()

	if _, err := f.Write(content); err != nil {
		return fmt.Errorf("write file: %w", err)
	}
	if _, err := f.WriteString("\n"); err != nil {
		return fmt.Errorf("write newline: %w", err)
	}

	wt, err := repo.Worktree()
	if err != nil {
		return fmt.Errorf("get worktree: %w", err)
	}

	if _, err := wt.Add(filename); err != nil {
		return fmt.Errorf("git add: %w", err)
	}

	_, err = wt.Commit(message, &git.CommitOptions{
		Author: &object.Signature{
			Name:  "gittool-bot",
			Email: "gittool@example.com",
			When:  time.Now(),
		},
	})
	if err != nil && err.Error() != "nothing to commit, working tree clean" {
		return fmt.Errorf("git commit: %w", err)
	}

	pushOpts := &git.PushOptions{RemoteName: "origin"}
	if username != "" && password != "" {
		pushOpts.Auth = &http.BasicAuth{Username: username, Password: password}
	}

	if err := repo.Push(pushOpts); err != nil && err != git.NoErrAlreadyUpToDate {
		return fmt.Errorf("git push: %w", err)
	}

	return nil
}
