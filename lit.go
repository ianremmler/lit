package lit

import (
	"errors"
	"os"
	"os/user"
	"sort"
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

func (l *Lit) Init() error {
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
	defer issueFile.Close()
	err = l.issues.Write(issueFile)
	if err != nil {
		return err
	}
	return nil
}

func (l *Lit) IssueIds() []string {
	if l.issues == nil {
		return []string{}
	}
	issueIds := []string{}
	for _, k := range l.issues.Kids() {
		if issue, ok := k.(*dgrl.Branch); ok {
			issueIds = append(issueIds, issue.Key())
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
	issue.Append(dgrl.NewLongLeaf("description", ""))
	l.issues.Append(issue)

	return id, nil
}

func (l *Lit) Issue(id string) *dgrl.Branch {
	for _, k := range l.issues.Kids() {
		if issue, ok := k.(*dgrl.Branch); ok && strings.HasPrefix(issue.Key(), id) {
			return issue
		}
	}
	return nil
}

func (l *Lit) Match(key, val string, doesMatch bool) []string {
	matches := []string{}
	for _, k := range l.issues.Kids() {
		if issue, ok := k.(*dgrl.Branch); ok {
			if contains(issue, key, val) == doesMatch {
				matches = append(matches, issue.Key())
			}
		}
	}
	return matches
}

type sorter struct{ ids, vals []string }

func newSorter(ids []string) *sorter {
	return &sorter{ids: ids, vals: make([]string, len(ids))}
}

func (s *sorter) Len() int { return len(s.ids) }

func (s *sorter) Less(i, j int) bool { return s.vals[i] < s.vals[j] }

func (s *sorter) Swap(i, j int) {
	s.ids[i], s.ids[j] = s.ids[j], s.ids[i]
	s.vals[i], s.vals[j] = s.vals[j], s.vals[i]
}

func (l *Lit) Sort(key string, ids []string, doAscend bool) {
	srt := newSorter(ids)
	for i := range ids {
		if issue := l.Issue(ids[i]); issue != nil {
			if val, ok := Get(issue, key); ok {
				srt.vals[i] = val
			}
		}
	}
	if doAscend {
		sort.Stable(srt)
	} else {
		sort.Stable(sort.Reverse(srt))
	}
}

func (l *Lit) Compare(key, val string, isLess bool) []string {
	matches := []string{}
	if val == "" {
		return matches
	}
	for _, k := range l.issues.Kids() {
		if issue, ok := k.(*dgrl.Branch); ok {
			if compare(issue, key, val, isLess) == isLess {
				matches = append(matches, issue.Key())
			}
		}
	}
	return matches
}

func Get(issue *dgrl.Branch, key string) (string, bool) {
	if issue == nil {
		return "", false
	}
	for _, k := range issue.Kids() {
		if leaf, ok := k.(*dgrl.Leaf); ok {
			if strings.HasPrefix(leaf.Key(), key) {
				return leaf.Value(), true
			}
		}
	}
	return "", false
}

func Set(issue *dgrl.Branch, key, val string) bool {
	if issue == nil {
		return false
	}
	for _, k := range issue.Kids() {
		if leaf, ok := k.(*dgrl.Leaf); ok {
			if strings.HasPrefix(leaf.Key(), key) {
				leaf.SetValue(val)
				return true
			}
		}
	}
	return false
}

func contains(issue *dgrl.Branch, key, val string) bool {
	if issue == nil {
		return false
	}
	if key == "comment" {
		return commentContains(issue, val)
	}
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

func commentContains(issue *dgrl.Branch, val string) bool {
	if issue == nil {
		return false
	}
	for _, k := range issue.Kids() {
		if comment, ok := k.(*dgrl.Branch); ok {
			for _, kk := range comment.Kids() {
				if leaf, ok := kk.(*dgrl.Leaf); ok {
					if strings.Contains(leaf.Value(), val) {
						return true
					}
				}
			}
		}
	}
	return false
}

func compare(issue *dgrl.Branch, key, val string, isLess bool) bool {
	if issue == nil {
		return false
	}
	if key == "comment" {
		return commentCompare(issue, val, isLess)
	}
	issueVal, ok := Get(issue, key)
	if !ok || issueVal == "" {
		return !isLess
	}
	if isLess {
		return issueVal < val
	}
	return issueVal <= val
}

func commentCompare(issue *dgrl.Branch, time string, isLess bool) bool {
	if issue == nil {
		return false
	}
	for _, k := range issue.Kids() {
		if comment, ok := k.(*dgrl.Branch); ok {
			commentTime := comment.Key()
			if isLess {
				return commentTime < time
			}
			return commentTime <= time
		}
	}
	return !isLess
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
