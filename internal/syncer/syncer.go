package syncer

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/agentspack/agentspack/internal/wizard"
)

// SyncReposFile is the name of the file containing the list of repos to sync
const SyncReposFile = "sync_repos.md"

// Syncer handles syncing generated files to GitHub repositories
type Syncer struct {
	config    *wizard.Config
	outputDir string
	repos     []string
}

// SyncResult represents the result of syncing to a single repository
type SyncResult struct {
	Repo    string
	Success bool
	Message string
	PRURL   string // Only set if a PR was created
}

// New creates a new Syncer instance
func New(config *wizard.Config, outputDir string) *Syncer {
	return &Syncer{
		config:    config,
		outputDir: outputDir,
	}
}

// Run executes the sync process for all configured repositories
func (s *Syncer) Run() error {
	// Check prerequisites
	if err := CheckGHInstalled(); err != nil {
		return err
	}

	if err := CheckGHAuth(); err != nil {
		return err
	}

	// Load repositories from sync_repos.md
	repos, err := s.loadRepos()
	if err != nil {
		return fmt.Errorf("failed to load repositories: %w", err)
	}

	if len(repos) == 0 {
		fmt.Println("No repositories found in " + SyncReposFile)
		return nil
	}

	s.repos = repos

	fmt.Printf("\nSyncing to %d GitHub repositories...\n\n", len(repos))

	// Generate a unique branch name with timestamp
	branchName := fmt.Sprintf("agentspack/update-%s", time.Now().Format("20060102-150405"))

	var results []SyncResult
	successCount := 0
	skipCount := 0
	failCount := 0

	for _, repo := range repos {
		fmt.Printf("  → %s ... ", repo)

		result := s.syncRepo(repo, branchName)
		results = append(results, result)

		if result.Success {
			if result.PRURL != "" {
				fmt.Printf("PR created: %s\n", result.PRURL)
				successCount++
			} else if strings.Contains(result.Message, "no changes") {
				fmt.Printf("skipped (no changes)\n")
				skipCount++
			} else if strings.Contains(result.Message, "merged") {
				fmt.Printf("merged to %s\n", s.config.TargetBranch)
				successCount++
			} else if strings.Contains(result.Message, "initialized") {
				fmt.Printf("%s\n", result.Message)
				successCount++
			} else {
				fmt.Printf("%s\n", result.Message)
				successCount++
			}
		} else {
			fmt.Printf("failed: %s\n", result.Message)
			failCount++
		}
	}

	// Print summary
	fmt.Println()
	if s.config.SyncMode == wizard.SyncModePR {
		fmt.Printf("Sync complete! %d PRs created", successCount)
	} else {
		fmt.Printf("Sync complete! %d repos updated", successCount)
	}
	if skipCount > 0 {
		fmt.Printf(", %d skipped", skipCount)
	}
	if failCount > 0 {
		fmt.Printf(", %d failed", failCount)
	}
	fmt.Println()

	if failCount > 0 {
		return fmt.Errorf("%d repositories failed to sync", failCount)
	}

	return nil
}

// loadRepos reads the sync_repos.md file and returns a list of repositories
func (s *Syncer) loadRepos() ([]string, error) {
	file, err := os.Open(SyncReposFile)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var repos []string
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Validate repo format (owner/repo)
		if !isValidRepoFormat(line) {
			fmt.Printf("Warning: invalid repo format '%s', skipping\n", line)
			continue
		}

		repos = append(repos, line)
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return repos, nil
}

// isValidRepoFormat checks if the repo string is in owner/repo format
func isValidRepoFormat(repo string) bool {
	parts := strings.Split(repo, "/")
	if len(parts) != 2 {
		return false
	}
	// Both owner and repo should be non-empty and not contain spaces
	return len(parts[0]) > 0 && len(parts[1]) > 0 &&
		!strings.Contains(parts[0], " ") && !strings.Contains(parts[1], " ")
}

// syncRepo syncs the generated files to a single repository
func (s *Syncer) syncRepo(repo, branchName string) SyncResult {
	// Create a temp directory for cloning
	tempDir, err := os.MkdirTemp("", "agentspack-sync-*")
	if err != nil {
		return SyncResult{Repo: repo, Success: false, Message: fmt.Sprintf("failed to create temp dir: %v", err)}
	}
	defer os.RemoveAll(tempDir)

	repoDir := filepath.Join(tempDir, "repo")

	// Clone the repository
	if err := CloneRepo(repo, repoDir); err != nil {
		return SyncResult{Repo: repo, Success: false, Message: fmt.Sprintf("clone failed: %v", err)}
	}

	// Check if repo is empty (no commits yet)
	isEmptyRepo := IsEmptyRepo(repoDir)

	// Create and checkout new branch (handles empty repos specially)
	if err := CreateBranch(repoDir, branchName, s.config.TargetBranch); err != nil {
		return SyncResult{Repo: repo, Success: false, Message: fmt.Sprintf("branch creation failed: %v", err)}
	}

	// Copy generated files to the repo
	if err := s.copyFiles(repoDir); err != nil {
		return SyncResult{Repo: repo, Success: false, Message: fmt.Sprintf("copy failed: %v", err)}
	}

	// Check if there are any changes (for empty repos, there will always be changes)
	hasChanges, err := HasChanges(repoDir)
	if err != nil {
		return SyncResult{Repo: repo, Success: false, Message: fmt.Sprintf("diff check failed: %v", err)}
	}

	if !hasChanges {
		return SyncResult{Repo: repo, Success: true, Message: "no changes"}
	}

	// Commit and push
	commitMsg := "chore: update AI agent configurations\n\nGenerated by agentspack"
	if err := CommitAndPush(repoDir, commitMsg); err != nil {
		return SyncResult{Repo: repo, Success: false, Message: fmt.Sprintf("commit/push failed: %v", err)}
	}

	// For empty repos, we can't create a PR (no base branch to compare against)
	// Just push directly to the target branch
	if isEmptyRepo {
		return SyncResult{Repo: repo, Success: true, Message: fmt.Sprintf("initialized %s branch", s.config.TargetBranch)}
	}

	// Create PR or merge directly
	if s.config.SyncMode == wizard.SyncModePR {
		prURL, err := CreatePR(repo, branchName, s.config.TargetBranch)
		if err != nil {
			return SyncResult{Repo: repo, Success: false, Message: fmt.Sprintf("PR creation failed: %v", err)}
		}
		return SyncResult{Repo: repo, Success: true, PRURL: prURL}
	}

	// Direct merge mode
	if err := MergeBranch(repoDir, branchName, s.config.TargetBranch); err != nil {
		return SyncResult{Repo: repo, Success: false, Message: fmt.Sprintf("merge failed: %v", err)}
	}

	return SyncResult{Repo: repo, Success: true, Message: "merged"}
}

// copyFiles copies all generated provider folders to the target repo
func (s *Syncer) copyFiles(repoDir string) error {
	// Read the output directory to find all provider folders
	entries, err := os.ReadDir(s.outputDir)
	if err != nil {
		return fmt.Errorf("failed to read output directory: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			// Copy top-level files (like CLAUDE.md, AGENTS.md)
			srcPath := filepath.Join(s.outputDir, entry.Name())
			dstPath := filepath.Join(repoDir, entry.Name())
			if err := copyFile(srcPath, dstPath); err != nil {
				return fmt.Errorf("failed to copy %s: %w", entry.Name(), err)
			}
			continue
		}

		// Copy provider directories (cursor, claude-code, codex)
		srcDir := filepath.Join(s.outputDir, entry.Name())
		dstDir := filepath.Join(repoDir, entry.Name())

		if err := copyDir(srcDir, dstDir); err != nil {
			return fmt.Errorf("failed to copy %s: %w", entry.Name(), err)
		}
	}

	return nil
}

// copyFile copies a single file from src to dst
func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}

	// Ensure parent directory exists
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}

	return os.WriteFile(dst, data, 0644)
}

// copyDir recursively copies a directory from src to dst
func copyDir(src, dst string) error {
	// Get source directory info
	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}

	// Create destination directory
	if err := os.MkdirAll(dst, srcInfo.Mode()); err != nil {
		return err
	}

	// Read source directory
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			if err := copyDir(srcPath, dstPath); err != nil {
				return err
			}
		} else {
			if err := copyFile(srcPath, dstPath); err != nil {
				return err
			}
		}
	}

	return nil
}
