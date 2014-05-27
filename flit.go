package flit

import (
	"errors"
	"fmt"
	"os"
	"os/user"
	"strings"
	"time"

	"code.google.com/p/go-uuid/uuid"
	"github.com/ianremmler/dgrl"
)

type Flit struct {
	issues *dgrl.Branch
}

func New() *Flit {
	return &Flit{issues: dgrl.NewRoot()}
}

func (f *Flit) InitFile() error {
	issueFile, err := os.OpenFile("issues", os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		return err
	}
	issueFile.Close()
	return nil
}

func (f *Flit) Load() error {
	issueFile, err := os.Open("issues")
	if err != nil {
		return err
	}
	defer issueFile.Close()
	issues := dgrl.NewParser().Parse(issueFile)
	if issues == nil {
		return errors.New("error parsing issue file")
	}
	f.issues = issues
	return nil
}

func (f *Flit) Store() error {
	if f.issues == nil {
		return errors.New("issues not initialized")
	}
	issueFile, err := os.Create("issues")
	if err != nil {
		return err
	}
	fmt.Fprintln(issueFile, f.issues)
	issueFile.Close()
	return nil
}

func (f *Flit) AppendIssues() error {
	if f.issues == nil {
		return errors.New("issues not initialized")
	}
	issueFile, err := os.OpenFile("issues", os.O_RDWR|os.O_APPEND, 0666)
	if err != nil {
		return err
	}
	fmt.Fprintln(issueFile, f.issues)
	issueFile.Close()
	return nil
}

func (f *Flit) IssueIds() []string {
	if f.issues == nil {
		return []string{}
	}
	issueIds := []string{}
	for _, node := range f.issues.Kids() {
		if node.Type() == dgrl.BranchType {
			issueIds = append(issueIds, node.Key())
		}
	}
	return issueIds
}

func (f *Flit) NewIssue() (string, error) {
	if f.issues == nil {
		return "", errors.New("issues not initialized")
	}
	id := uuid.New()
	issue := dgrl.NewBranch(id)
	issue.Append(dgrl.NewLeaf("created", Stamp()))
	issue.Append(dgrl.NewLeaf("updated", Stamp()))
	issue.Append(dgrl.NewLeaf("closed", ""))
	issue.Append(dgrl.NewLeaf("summary", ""))
	issue.Append(dgrl.NewLeaf("tags", ""))
	issue.Append(dgrl.NewLeaf("status", ""))
	issue.Append(dgrl.NewLeaf("priority", ""))
	issue.Append(dgrl.NewLeaf("assigned", ""))
	issue.Append(dgrl.NewLongLeaf("description", "\n"))
	f.issues.Append(issue)

	return id, nil
}

func (f *Flit) Issue(id string) *dgrl.Branch {
	for _, node := range f.issues.Kids() {
		if node.Type() == dgrl.BranchType && node.Key() == id {
			return node.(*dgrl.Branch)
		}
	}
	return nil
}

func (f *Flit) Match(key, val string, doesMatch bool) []string {
	matches := []string{}
	for _, node := range f.issues.Kids() {
		if issue, ok := node.(*dgrl.Branch); ok {
			if IssueContains(issue, key, val) == doesMatch {
				matches = append(matches, issue.Key())
			}
		}
	}
	return matches
}

func Get(issue *dgrl.Branch, key string) (string, bool) {
	for _, kid := range issue.Kids() {
		if leaf, ok := kid.(*dgrl.Leaf); ok {
			if strings.Contains(leaf.Key(), key) {
				return leaf.Value(), true
			}
		}
	}
	return "", false
}

func IssueContains(issue *dgrl.Branch, key, val string) bool {
	if issueVal, ok := Get(issue, key); ok {
		if strings.Contains(issueVal, val) {
			return true
		}
	}
	return false
}

func Set(issue *dgrl.Branch, key, val string) bool {
	for _, kid := range issue.Kids() {
		if leaf, ok := kid.(*dgrl.Leaf); ok {
			if strings.Contains(leaf.Key(), key) {
				leaf.SetValue(val)
				return true
			}
		}
	}
	return false
}

func curTime() string {
	return time.Now().Format(time.RFC3339)
}

func curUser() (string, error) {
	user, err := user.Current()
	if err != nil {
		return "", err
	}
	return user.Username, nil
}

func Stamp() string {
	user, err := curUser()
	if err != nil {
		user = "?"
	}
	return user + " " + curTime()
}
