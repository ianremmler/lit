// Package lit provides the core of the lightweight issue tracker.
package lit

import (
	"errors"
	"fmt"
	"os"
	"os/user"
	"sort"
	"strings"
	"time"

	"code.google.com/p/go-uuid/uuid"
	"github.com/ianremmler/dgrl"
)

const (
	issueFilename = "issues"
)

var (
	username = "?"
)

func init() {
	if env := os.Getenv("LIT_USER"); env != "" {
		username = env
	} else if user, err := user.Current(); err == nil {
		username = user.Username
	}
}

// Stamp returns a string consisting of the current time in RFC3339 UTC format
// and the username, separated by a space.
func Stamp() string {
	return fmt.Sprintf("%s %s", time.Now().UTC().Format(time.RFC3339), username)
}

// Get returns the value for the given key, if found in the issue.
// key may be a substring matching the beginning of the issue key.
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

// Set sets the value for the given key, if found in the issue.
// key may be a substring matching the beginning of the issue key.
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

// Lit stores and manipulates issues
type Lit struct {
	issues *dgrl.Branch
}

// New constructs a new Lit.
func New() *Lit {
	return &Lit{issues: dgrl.NewRoot()}
}

// Init initializes the issue tracker.
func (l *Lit) Init() error {
	issueFile, err := os.OpenFile(issueFilename, os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		return err
	}
	issueFile.Close()
	return nil
}

// Load parses the issue file and populates the list of issues
func (l *Lit) Load() error {
	issueFile, err := os.Open(issueFilename)
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

// Store writes the issue list to the file
func (l *Lit) Store() error {
	if l.issues == nil {
		return errors.New("issues not initialized")
	}
	issueFile, err := os.Create(issueFilename)
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

// IssueIds returns a slice of all issue ids
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

// NewIssue adds a new issue and returns its id
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
	issue.Append(dgrl.NewLeaf("priority", ""))
	issue.Append(dgrl.NewLeaf("assigned", ""))
	issue.Append(dgrl.NewLongLeaf("description", ""))
	l.issues.Append(issue)

	return id, nil
}

// Issue returns an issue for the given id
func (l *Lit) Issue(id string) *dgrl.Branch {
	for _, k := range l.issues.Kids() {
		if issue, ok := k.(*dgrl.Branch); ok && strings.HasPrefix(issue.Key(), id) {
			return issue
		}
	}
	return nil
}

// Match returns a list of ids for all issues whose value for key contains val.
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

// Len returns the number of elements to sort.
func (s *sorter) Len() int { return len(s.ids) }

// Less returns whether the first element is less than the second.
func (s *sorter) Less(i, j int) bool { return s.vals[i] < s.vals[j] }

// Swap swaps two elements
func (s *sorter) Swap(i, j int) {
	s.ids[i], s.ids[j] = s.ids[j], s.ids[i]
	s.vals[i], s.vals[j] = s.vals[j], s.vals[i]
}

// Sort sorts the list of ids by the value for the given key.
func (l *Lit) Sort(ids []string, key string, doAscend bool) {
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

// Compare returns a list of ids for all issues whose value for key is less
// or greater, determined by isLess, than val.
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

func contains(issue *dgrl.Branch, key, val string) bool {
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

// ModifyTag adds or removes a tag for a given issue
func ModifyTag(issue *dgrl.Branch, tag string, doAdd bool) bool {
	tags, _ := Get(issue, "tag")
	tagSet := tagStrToSet(tags)
	if doAdd {
		tagSet[tag] = struct{}{}
	} else {
		delete(tagSet, tag)
	}
	return Set(issue, "tag", setToTagStr(tagSet))
}

func tagStrToSet(tagStr string) map[string]struct{} {
	set := map[string]struct{}{}
	for _, tag := range strings.Fields(tagStr) {
		set[tag] = struct{}{}
	}
	return set
}

func setToTagStr(set map[string]struct{}) string {
	tags := make([]string, len(set))
	for tag := range set {
		tags = append(tags, tag)
	}
	return strings.TrimSpace(strings.Join(tags, " "))
}
