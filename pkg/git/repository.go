package git

import (
	"context"
	"fmt"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
)

// Repository represents a git repository wrapper
type Repository struct {
	Name        string
	Owner       string
	Host        string
	Branch      string
	Token       string
	AuthorName  string
	AuthorEmail string
}

// NewRepository returns a git repository wrapper
func NewRepository(name, owner, host, branch, token, authorName, authorEmail string) (*Repository, error) {
	if name == "" {
		return nil, fmt.Errorf("name required")
	}
	if owner == "" {
		return nil, fmt.Errorf("owner required")
	}
	if host == "" {
		return nil, fmt.Errorf("host required")
	}
	if branch == "" {
		return nil, fmt.Errorf("branch required")
	}
	if token == "" {
		return nil, fmt.Errorf("token required")
	}
	if authorName == "" {
		authorName = "tk"
	}
	if authorEmail == "" {
		authorEmail = "tk@users.noreply.git-scm.com"
	}

	return &Repository{
		Name:        name,
		Owner:       owner,
		Host:        host,
		Branch:      branch,
		Token:       token,
		AuthorName:  authorName,
		AuthorEmail: authorEmail,
	}, nil
}

// GetURL returns the repository HTTPS address
func (r *Repository) GetURL() string {
	return fmt.Sprintf("https://%s/%s/%s", r.Host, r.Owner, r.Name)
}

// GetSSH returns the repository SSH address
func (r *Repository) GetSSH() string {
	return fmt.Sprintf("ssh://git@%s/%s/%s", r.Host, r.Owner, r.Name)
}

func (r *Repository) auth() transport.AuthMethod {
	return &http.BasicAuth{
		Username: "git",
		Password: r.Token,
	}
}

// Checkout repository at specified path
func (r *Repository) Checkout(ctx context.Context, path string) (*git.Repository, error) {
	repo, err := git.PlainCloneContext(ctx, path, false, &git.CloneOptions{
		URL:           r.GetURL(),
		Auth:          r.auth(),
		RemoteName:    git.DefaultRemoteName,
		ReferenceName: plumbing.NewBranchReferenceName(r.Branch),
		SingleBranch:  true,
		NoCheckout:    false,
		Progress:      nil,
		Tags:          git.NoTags,
	})
	if err != nil {
		return nil, fmt.Errorf("git clone error: %w", err)
	}

	_, err = repo.Head()
	if err != nil {
		return nil, fmt.Errorf("git resolve HEAD error: %w", err)
	}

	return repo, nil
}

// Commit changes for the specified path, returns false if no changes are made
func (r *Repository) Commit(ctx context.Context, repo *git.Repository, path, message string) (bool, error) {
	w, err := repo.Worktree()
	if err != nil {
		return false, err
	}

	_, err = w.Add(path)
	if err != nil {
		return false, err
	}

	status, err := w.Status()
	if err != nil {
		return false, err
	}

	if !status.IsClean() {
		if _, err := w.Commit(message, &git.CommitOptions{
			Author: &object.Signature{
				Name:  r.AuthorName,
				Email: r.AuthorEmail,
				When:  time.Now(),
			},
		}); err != nil {
			return false, err
		}
		return true, nil
	}

	return false, nil
}

// Push commits to origin
func (r *Repository) Push(ctx context.Context, repo *git.Repository) error {
	err := repo.PushContext(ctx, &git.PushOptions{
		Auth:     r.auth(),
		Progress: nil,
	})
	if err != nil {
		return fmt.Errorf("git push error: %w", err)
	}
	return nil
}
