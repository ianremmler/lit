package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/ianremmler/dgrl"
	"github.com/ianremmler/flit"
)

const usage = `usage:

spec: . | all | <id(s)> | (with|without) <key> [<val>]
	'.' indicates the currently open issue

flit [help | usage]    Show usage
flit state [<spec>]    Show issue state
flit init              Initialize new issue tracker
flit new               Create new issue
flit id [<spec>]       List ids, optionally filtering by key/value
flit show [<spec>]     Show issue (default: current)
flit edit <id>         Edit issue`

var (
	args = os.Args[1:]
	it   = flit.New()
)

func main() {
	log.SetFlags(0)
	log.SetPrefix("flit: ")

	cmd := ""
	if len(args) > 0 {
		cmd = strings.ToLower(args[0])
		args = args[1:]
	}
	switch cmd {
	case "", "-h", "-help", "--help", "help", "-u", "-usage", "--usage", "usage":
		usageCmd()
	case "state":
		stateCmd()
	case "init":
		initCmd()
	case "new":
		newCmd()
	case "id":
		idCmd()
	case "show":
		showCmd()
	case "set":
		setCmd()
	case "edit":
		editCmd()
	default:
		log.Fatalln(cmd + " is not a valid command\n\n" + usage)
	}
}

func usageCmd() {
	fmt.Println(usage)
}

func initCmd() {
	if it.InitFile() != nil {
		log.Fatalln("init: Error initializing issue tracker")
	}
}

func newCmd() {
	id, err := it.NewIssue()
	checkErr("new", err)
	err = it.AppendIssues()
	checkErr("new", err)
	fmt.Println(id)
}

func stateCmd() {
	// verifyRepo()
	// for _, id := range specIds(args) {
	// fmt.Println(stateSummary(id))
	// }
}

func idCmd() {
	loadIssues("id")
	for _, id := range specIds(args) {
		if it.Issue(id) != nil {
			fmt.Println(id)
		}
	}
}

func showCmd() {
	loadIssues("show")
	for _, id := range specIds(args) {
		fmt.Println(it.Issue(id))
	}
}

func setCmd() {
	loadIssues("set")
	if len(args) < 3 {
		log.Fatalln("set: You must specify a valid issue id, key, and value")
	}
	id, key, val := args[0], args[1], args[2]
	issue := it.Issue(id)
	if issue == nil {
		log.Fatalln("set: Error finding issue")
	}
	flit.Set(issue, key, val)
	storeIssues("set")
}

func editCmd() {
	editor := os.Getenv("VISUAL")
	if editor == "" {
		editor = os.Getenv("EDITOR")
	}
	if editor == "" {
		log.Fatalln("VISUAL or EDITOR environment variable must be set")
	}
	if len(args) < 1 {
		log.Fatalln("edit: You must specify a valid issue id")
	}

	// get the issue
	loadIssues("edit")
	id := args[0]
	issue := it.Issue(id)
	if issue == nil {
		log.Fatalln("edit: Error finding issue")
	}

	// write the issue to a temp file
	filename := os.TempDir() + "/issue-" + id
	issueFile, err := os.Create(filename)
	checkErr("edit", err)
	fmt.Fprintln(issueFile, issue)
	issueFile.Close()

	// get original file state
	origStat, err := os.Stat(filename)
	checkErr("edit", err)

	// launch editor
	cmd := exec.Command(editor, filename)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	err = cmd.Run()
	checkErr("edit", err)

	// get updated file state, compare to original
	newStat, err := os.Stat(filename)
	checkErr("edit", err)
	if newStat.ModTime() == origStat.ModTime() {
		log.Fatalln("edit: Issue unchanged")
	}

	// parse issue from temp file
	issueFile, err = os.Open(filename)
	checkErr("edit", err)
	defer issueFile.Close()
	edited := dgrl.NewParser().Parse(issueFile)
	if edited == nil {
		log.Fatalln("edit: Error parsing issue")
	}

	// update issue if we find a match
	didUpdate := false
	for _, node := range edited.Kids() {
		if node.Type() == dgrl.BranchType && node.Key() == id {
			if editedIssue, ok := node.(*dgrl.Branch); ok {
				*issue = *editedIssue
				didUpdate = true
				break
			}
		}
	}
	if !didUpdate {
		log.Fatalln("edit: Error updating issue")
	}

	storeIssues("edit")
}

// func stateSummary(id string) string {
// verifyIssue(id)
// status, _ := it.Value(id, "status")
// typ, _ := it.Value(id, "type")
// priority, _ := it.Value(id, "priority")
// assigned, _ := it.Value(id, "assigned")
// summary, _ := it.Value(id, "summary")
// numAtt := len(it.Attachments(id))
// return fmt.Sprintf("%s %-7.7s %-7.7s %-7.7s %-7.7s %-2d %s",
// id, status, typ, priority, assigned, numAtt, summary)
// }

func matchIds(kv []string, doesMatch bool) []string {
	key, val := "", ""
	if len(kv) > 0 {
		key = kv[0]
	}
	if len(kv) > 1 {
		val = kv[1]
	}
	return it.Match(key, val, doesMatch)
}

func specIds(args []string) []string {
	ids := []string{}
	switch {
	case len(args) == 0:
		return ids
	case args[0] == "with":
		ids = matchIds(args[1:], true)
	case args[0] == "without":
		ids = matchIds(args[1:], false)
	case args[0] == "all":
		ids = it.IssueIds()
	default:
		ids = args
	}
	return ids
}

func loadIssues(cmd string) {
	err := it.Load()
	checkErr(cmd, err)
}

func storeIssues(cmd string) {
	err := it.Store()
	checkErr(cmd, err)
}

func checkErr(cmd string, err error) {
	if err != nil {
		log.Fatalf("%s: %s\n", cmd, err)
	}
}
