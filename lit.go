package lit

import (
	"errors"
	"os"
	"os/user"
	"strings"
	"time"

	"code.google.com/p/go-uuid/uuid"
	"github.com/ianremmler/dgrl"
)

type Lit struct {
	issues *dgrl.Branch
}

func New() *Lit {
	return &Lit{issues: dgrl.NewRoot()}
}

func (l *Lit) InitFile() error {
	issueFile, err := os.OpenFile("issues", os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		return err
	}
	issueFile.Close()
	return nil
}

func (l *Lit) Load() error {
	issueFile, err := os.Open("issues")
	if err != nil {
		return err
	}
	defer issueFile.Close()
	issues := dgrl.NewParser().Parse(issueFile)
	if issues == nil {
		return errors.New("error parsing issue file")
	}
	l.issues = issues
	return nil
}

func (l *Lit) Store() error {
	if l.issues == nil {
		return errors.New("issues not initialized")
	}
	issueFile, err := os.Create("issues")
	if err != nil {
		return err
	}

	l.issues.Write(issueFile)
	issueFile.Close()
	return nil
}

func (l *Lit) IssueIds() []string {
	if l.issues == nil {
		return []string{}
	}
	issueIds := []string{}
	for _, node := range l.issues.Kids() {
		if node.Type() == dgrl.BranchType {
			issueIds = append(issueIds, node.Key())
		}
	}
	return issueIds
}

func (l *Lit) NewIssue() (string, error) {
	if l.issues == nil {
		return "", errors.New("issues not initialized")
	}
	id := uuid.New()
	issue := dgrl.NewBranch(id)
	stamp := Stamp()
	issue.Append(dgrl.NewLeaf("created", stamp))
	issue.Append(dgrl.NewLeaf("updated", stamp))
	issue.Append(dgrl.NewLeaf("closed", ""))
	issue.Append(dgrl.NewLeaf("summary", ""))
	issue.Append(dgrl.NewLeaf("tags", ""))
	issue.Append(dgrl.NewLeaf("priority", "1"))
	issue.Append(dgrl.NewLeaf("assigned", ""))
	issue.Append(dgrl.NewLongLeaf("description", "\n"))
	l.issues.Append(issue)

	return id, nil
}

func (l *Lit) Issue(id string) *dgrl.Branch {
	for _, node := range l.issues.Kids() {
		if node.Type() == dgrl.BranchType && strings.HasPrefix(node.Key(), id) {
			return node.(*dgrl.Branch)
		}
	}
	return nil
}

func (l *Lit) Match(key, val string, doesMatch bool) []string {
	matches := []string{}
	for _, node := range l.issues.Kids() {
		if issue, ok := node.(*dgrl.Branch); ok {
			if Contains(issue, key, val) == doesMatch {
				matches = append(matches, issue.Key())
			}
		}
	}
	return matches
}

func (l *Lit) Compare(key, val string, isLess bool) []string {
	matches := []string{}
	if val == "" {
		return matches
	}
	for _, node := range l.issues.Kids() {
		if issue, ok := node.(*dgrl.Branch); ok {
			if Less(issue, key, val, !isLess) == isLess {
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

func Contains(issue *dgrl.Branch, key, val string) bool {
	if issueVal, ok := Get(issue, key); ok {
		if val == "" && issueVal == "" {
			return false
		}
		if strings.Contains(issueVal, val) {
			return true
		}
	}
	return false
}

func Less(issue *dgrl.Branch, key, val string, incl bool) bool {
	if issueVal, ok := Get(issue, key); ok {
		if issueVal == "" {
			return false
		}
		if incl {
			return issueVal <= val
		} else {
			return issueVal < val
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
	return time.Now().UTC().Format(time.RFC3339)
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
	return curTime() + " " + user
}
