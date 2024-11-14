package git

import (
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"golang.org/x/mod/module"
)

// Reference is a reference to either a commit/branch/partial/tag, or an annotated tag
type Reference struct {
	ref          *plumbing.Reference
	annotatedTag *object.Tag
}

func NewReference(ref *plumbing.Reference) *Reference {
	return &Reference{ref: ref, annotatedTag: nil}
}

func NewReferenceToAnnotatedTag(tag *object.Tag) *Reference {
	return &Reference{ref: nil, annotatedTag: tag}
}

func (r *Reference) Commit(repo *git.Repository) (*object.Commit, error) {
	if r.annotatedTag != nil {
		return r.annotatedTag.Commit()
	}

	return repo.CommitObject(r.ref.Hash())
}

func (r *Reference) EscapeHash() (string, error) {
	if r.annotatedTag != nil {
		return module.EscapeVersion(r.annotatedTag.Hash.String())
	}

	return module.EscapeVersion(r.ref.Hash().String())
}
