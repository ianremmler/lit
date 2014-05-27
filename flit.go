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
	defer issueFile.Close()
	if err != nil {
		return err
	}
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
	issue.Append(dgrl.NewLeaf("created", stamp()))
	issue.Append(dgrl.NewLeaf("updated", stamp()))
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

// func (f *Flit) SetWorkingValue(key, val string) bool {
// data, err := ioutil.ReadFile(f.issueFilename)
// if err != nil {
// return false
// }
// parser := dgrl.NewParser()
// tree := parser.Parse(strings.NewReader(string(data)))
// for _, node := range tree.Kids() {
// if leaf, ok := node.(*dgrl.Leaf); ok {
// if leaf.Key() == key {
// leaf.SetValue(val)
// issueFile, err := os.Create(f.issueFilename)
// if err != nil {
// return false
// }
// issueFile.WriteString(tree.String())
// issueFile.Close()
// return true
// }
// }
// }
// return false
// }

func (f *Flit) Set(id, key, val string, doesMatch bool) []string {
	matches := []string{}

	for _, node := range f.issues.Kids() {
		if issue, ok := node.(*dgrl.Branch); ok {
			if issueContains(issue, key, val) {
				matches = append(matches, issue.Key())
			}
		}
	}
	return matches
}

func (f *Flit) Match(key, val string, doesMatch bool) []string {
	matches := []string{}

	for _, node := range f.issues.Kids() {
		if issue, ok := node.(*dgrl.Branch); ok {
			if issueContains(issue, key, val) {
				matches = append(matches, issue.Key())
			}
		}
	}
	return matches
}

func issueContains(issue *dgrl.Branch, key, val string) bool {
	for _, kid := range issue.Kids() {
		if leaf, ok := kid.(*dgrl.Leaf); ok {
			if strings.Contains(leaf.Key(), key) && strings.Contains(leaf.Value(), val) {
				return true
			}
		}
	}
	return false
}

// if key == "" {
// return true
// }
// parser := dgrl.NewParser()
// tree := parser.Parse(strings.NewReader(f.IssueText(id)))
// for _, node := range tree.Kids() {
// if leaf, ok := node.(*dgrl.Leaf); ok {
// if strings.Contains(leaf.Key(), key) && strings.Contains(leaf.Value(), val) {
// return true
// }
// }
// }
// return false
// }

// func (f *Flit) ToDgrl(ids []string) *dgrl.Branch {
// parser := dgrl.NewParser()
// root := dgrl.NewRoot()
// for _, id := range ids {
// issue := dgrl.NewBranch(FormatId(id))
// tree := parser.Parse(strings.NewReader(f.IssueText(id)))
// for _, kid := range tree.Kids() {
// issue.Append(kid)
// }
// root.Append(issue)
// }
// return root
// }

// func (f *Flit) Attachments(id string) []string {
// repo := gitgo.New()
// branch := ""
// if id != "" {
// branch = f.IdToBranch(id)
// }
// att, err := repo.Run("diff", "--name-only", "--diff-filter=A", "master", branch)
// if err != nil {
// return []string{}
// }
// attList := strings.Split(strings.TrimSpace(att), "\n")
// // Split returns slice with 1 empty string on empty input, so just return empty slice
// if len(attList) == 1 && attList[0] == "" {
// return []string{}
// }
// return attList
// }

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

func stamp() string {
	user, err := curUser()
	if err != nil {
		user = "?"
	}
	return user + " " + curTime()
}
