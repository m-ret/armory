package registry

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/m-ret/armory/internal/config"
	"github.com/m-ret/armory/internal/teams"
)

// SearchResult represents a skill found in a preset that has a remote source.
type SearchResult struct {
	Name       string
	Collection string
	RepoURL    string
}

// InstallResult reports the outcome of installing a single skill.
type InstallResult struct {
	Name   string
	Method string // "npx" or "git-clone"
	Dir    string // where it was installed
	Err    error
}

// ProgressFunc is a callback for reporting installation progress.
type ProgressFunc func(msg string)

// DefaultInstallDir returns the default directory where skills are installed.
func DefaultInstallDir() string {
	return config.ExpandPath("~/.agents/skills")
}

// SearchByNames looks up each name in the preset skill lists.
// Skills found in presets with a Collection+RepoURL go into found.
// Everything else (known but no collection, or entirely unknown) goes into notFound.
func SearchByNames(names []string) (found []SearchResult, notFound []string) {
	presets := teams.AllPresets()

	for _, name := range names {
		var matched bool
		for _, p := range presets {
			if containsSkill(p.Skills, name) {
				if p.Collection != "" && p.RepoURL != "" {
					found = append(found, SearchResult{
						Name:       name,
						Collection: p.Collection,
						RepoURL:    p.RepoURL,
					})
				} else {
					notFound = append(notFound, name)
				}
				matched = true
				break
			}
		}
		if !matched {
			notFound = append(notFound, name)
		}
	}
	return found, notFound
}

// HasNpx reports whether npx is available on the PATH.
func HasNpx() bool {
	_, err := exec.LookPath("npx")
	return err == nil
}

// InstallSkills installs the given search results, grouping by collection.
// Skills that already exist in DefaultInstallDir are skipped.
func InstallSkills(ctx context.Context, results []SearchResult, progress ProgressFunc) ([]InstallResult, error) {
	installDir := DefaultInstallDir()

	// Group results by collection.
	groups := make(map[string][]SearchResult)
	var order []string
	for _, r := range results {
		if _, seen := groups[r.Collection]; !seen {
			order = append(order, r.Collection)
		}
		groups[r.Collection] = append(groups[r.Collection], r)
	}

	var allResults []InstallResult

	for _, coll := range order {
		members := groups[coll]

		// Check which skills already exist.
		var needed []SearchResult
		for _, m := range members {
			dest := filepath.Join(installDir, m.Name)
			if _, err := os.Stat(dest); err == nil {
				progress(fmt.Sprintf("Skill %q already installed, skipping", m.Name))
				allResults = append(allResults, InstallResult{
					Name:   m.Name,
					Method: "existing",
					Dir:    dest,
				})
				continue
			}
			needed = append(needed, m)
		}
		if len(needed) == 0 {
			continue
		}

		repoURL := needed[0].RepoURL

		if HasNpx() {
			progress(fmt.Sprintf("Installing collection %q via npx", coll))
			cmd := exec.CommandContext(ctx, "npx", "skills", "add", coll, "-g")
			out, err := cmd.CombinedOutput()
			method := "npx"
			for _, m := range needed {
				dest := filepath.Join(installDir, m.Name)
				var skillErr error
				if err != nil {
					skillErr = fmt.Errorf("npx skills add failed: %s: %w", strings.TrimSpace(string(out)), err)
				}
				allResults = append(allResults, InstallResult{
					Name:   m.Name,
					Method: method,
					Dir:    dest,
					Err:    skillErr,
				})
			}
		} else if repoURL != "" {
			progress(fmt.Sprintf("Cloning %s (npx not available)", repoURL))
			results, err := cloneAndCopySkills(ctx, repoURL, needed, installDir, progress)
			if err != nil {
				return allResults, err
			}
			allResults = append(allResults, results...)
		}
	}

	return allResults, nil
}

// InstallCollection installs an entire collection. After cloning, it scans
// for subdirectories containing SKILL.md and copies each to DefaultInstallDir.
func InstallCollection(ctx context.Context, collection string, repoURL string, progress ProgressFunc) ([]InstallResult, error) {
	installDir := DefaultInstallDir()
	if err := os.MkdirAll(installDir, 0o755); err != nil {
		return nil, fmt.Errorf("creating install dir: %w", err)
	}

	if HasNpx() {
		progress(fmt.Sprintf("Installing collection %q via npx", collection))
		cmd := exec.CommandContext(ctx, "npx", "skills", "add", collection, "-g")
		out, err := cmd.CombinedOutput()
		if err != nil {
			return nil, fmt.Errorf("npx skills add %s failed: %s: %w", collection, strings.TrimSpace(string(out)), err)
		}

		// Scan install dir for what was just installed.
		var results []InstallResult
		entries, _ := os.ReadDir(installDir)
		for _, e := range entries {
			if e.IsDir() {
				results = append(results, InstallResult{
					Name:   e.Name(),
					Method: "npx",
					Dir:    filepath.Join(installDir, e.Name()),
				})
			}
		}
		return results, nil
	}

	if repoURL == "" {
		return nil, fmt.Errorf("no npx available and no repo URL for collection %q", collection)
	}

	progress(fmt.Sprintf("Cloning %s (npx not available)", repoURL))

	tmpDir, err := os.MkdirTemp("", "armory-collection-*")
	if err != nil {
		return nil, fmt.Errorf("creating temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	cmd := exec.CommandContext(ctx, "git", "clone", "--depth=1", repoURL, tmpDir)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("git clone failed: %s: %w", strings.TrimSpace(string(out)), err)
	}

	// Find all subdirs with SKILL.md.
	var results []InstallResult
	entries, err := os.ReadDir(tmpDir)
	if err != nil {
		return nil, fmt.Errorf("reading cloned dir: %w", err)
	}

	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		skillFile := filepath.Join(tmpDir, e.Name(), "SKILL.md")
		if _, err := os.Stat(skillFile); err != nil {
			continue
		}

		dest := filepath.Join(installDir, e.Name())
		if _, err := os.Stat(dest); err == nil {
			progress(fmt.Sprintf("Skill %q already installed, skipping", e.Name()))
			results = append(results, InstallResult{
				Name:   e.Name(),
				Method: "existing",
				Dir:    dest,
			})
			continue
		}

		progress(fmt.Sprintf("Copying skill %q", e.Name()))
		src := filepath.Join(tmpDir, e.Name())
		if err := copyDir(src, dest); err != nil {
			results = append(results, InstallResult{
				Name:   e.Name(),
				Method: "git-clone",
				Dir:    dest,
				Err:    fmt.Errorf("copying skill dir: %w", err),
			})
			continue
		}

		results = append(results, InstallResult{
			Name:   e.Name(),
			Method: "git-clone",
			Dir:    dest,
		})
	}

	return results, nil
}

// cloneAndCopySkills clones a repo and copies specific skill subdirs.
func cloneAndCopySkills(ctx context.Context, repoURL string, needed []SearchResult, installDir string, progress ProgressFunc) ([]InstallResult, error) {
	if err := os.MkdirAll(installDir, 0o755); err != nil {
		return nil, fmt.Errorf("creating install dir: %w", err)
	}

	tmpDir, err := os.MkdirTemp("", "armory-skills-*")
	if err != nil {
		return nil, fmt.Errorf("creating temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	cmd := exec.CommandContext(ctx, "git", "clone", "--depth=1", repoURL, tmpDir)
	out, err := cmd.CombinedOutput()
	if err != nil {
		// All skills in this group fail.
		var results []InstallResult
		cloneErr := fmt.Errorf("git clone failed: %s: %w", strings.TrimSpace(string(out)), err)
		for _, m := range needed {
			results = append(results, InstallResult{
				Name:   m.Name,
				Method: "git-clone",
				Dir:    filepath.Join(installDir, m.Name),
				Err:    cloneErr,
			})
		}
		return results, nil
	}

	var results []InstallResult
	for _, m := range needed {
		src := filepath.Join(tmpDir, m.Name)
		dest := filepath.Join(installDir, m.Name)

		if _, err := os.Stat(src); err != nil {
			results = append(results, InstallResult{
				Name:   m.Name,
				Method: "git-clone",
				Dir:    dest,
				Err:    fmt.Errorf("skill subdir %q not found in cloned repo", m.Name),
			})
			continue
		}

		progress(fmt.Sprintf("Copying skill %q", m.Name))
		if err := copyDir(src, dest); err != nil {
			results = append(results, InstallResult{
				Name:   m.Name,
				Method: "git-clone",
				Dir:    dest,
				Err:    fmt.Errorf("copying skill dir: %w", err),
			})
			continue
		}

		results = append(results, InstallResult{
			Name:   m.Name,
			Method: "git-clone",
			Dir:    dest,
		})
	}

	return results, nil
}

// copyDir recursively copies src to dst.
func copyDir(src, dst string) error {
	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(dst, srcInfo.Mode()); err != nil {
		return err
	}

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

// copyFile copies a single file from src to dst.
func copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	srcInfo, err := srcFile.Stat()
	if err != nil {
		return err
	}

	dstFile, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, srcInfo.Mode())
	if err != nil {
		return err
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	return err
}

// containsSkill checks if a skill name appears in a skill list (case-insensitive).
func containsSkill(skills []string, name string) bool {
	lower := strings.ToLower(name)
	for _, s := range skills {
		if strings.ToLower(s) == lower {
			return true
		}
	}
	return false
}
