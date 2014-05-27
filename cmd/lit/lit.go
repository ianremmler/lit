package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/ianremmler/dgrl"
	"github.com/ianremmler/lit"
)

const usage = `usage:

spec: . | all | <id(s)> | (with|without) <field> [<val>]
	'.' indicates the currently open issue

lit [help | usage]            Show usage
lit list <spec>               Show list of issues matching spec
lit init                      Initialize new issue tracker
lit new                       Create new issue
lit id <spec>                 List ids matching spec
lit show <spec>               Show issues matching spec
lit set <field> <val> <spec>  Set issue field
lit edit <id>                 Edit issue
lit close <spec>              Close issue matching spec
lit reopen <spec>             Reopen closed issues matching spec`

var (
	args = os.Args[1:]
	it   = lit.New()
)

func main() {
	log.SetFlags(0)
	log.SetPrefix("lit: ")

	cmd := ""
	if len(args) > 0 {
		cmd = strings.ToLower(args[0])
		args = args[1:]
	}
	switch cmd {
	case "", "-h", "-help", "--help", "help", "-u", "-usage", "--usage", "usage":
		usageCmd()
	case "list":
		listCmd()
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
	case "close":
		closeCmd()
	case "reopen":
		reopenCmd()
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

func listCmd() {
	loadIssues("list")
	fmt.Printf("%-36.36s %-8.8s %-8.8s %-8.8s %-8.8s %s\n",
		"id", "status", "tags", "priority", "assigned", "summary")
	for _, id := range specIds(args) {
		issue := it.Issue(id)
		if issue != nil {
			fmt.Println(listInfo(id, issue))
		}
	}
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
	if len(args) < 3 {
		log.Fatalln("set: You must specify a field, and value, and spec")
	}
	key, val, spec := args[0], args[1], args[2:]
	loadIssues("set")
	stamp := lit.Stamp()
	for _, id := range specIds(spec) {
		issue := it.Issue(id)
		if issue == nil {
			log.Fatalln("set: Error finding issue")
		}
		ok := lit.Set(issue, key, val)
		ok = ok && lit.Set(issue, "updated", stamp)
		if !ok {
			log.Fatalln("set: Error updating issue fields")
		}
	}
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
	id := args[0]
	loadIssues("edit")
	issue := it.Issue(id)
	if !lit.Set(issue, "updated", lit.Stamp()) {
		log.Fatalln("edit: Error setting update time")
	}
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

func closeCmd() {
	if len(args) < 1 {
		log.Fatalln("close: You must specify a spec to close")
	}
	stamp := lit.Stamp()
	for _, id := range specIds(args) {
		loadIssues("close")
		issue := it.Issue(id)
		if issue == nil {
			log.Fatalln("close: Error finding issue")
		}
		ok := lit.Set(issue, "status", "closed")
		ok = ok && lit.Set(issue, "closed", stamp)
		ok = ok && lit.Set(issue, "updated", stamp)
		if !ok {
			log.Fatalln("close: Error updating issue fields")
		}
	}
	storeIssues("close")
}

func reopenCmd() {
	if len(args) < 1 {
		log.Fatalln("reopen: You must specify an issue to reopen")
	}
	loadIssues("reopen")
	stamp := lit.Stamp()
	for _, id := range specIds(args) {
		issue := it.Issue(id)
		if issue == nil {
			log.Fatalln("reopen: Error finding issue")
		}
		ok := lit.Set(issue, "status", "open")
		ok = ok && lit.Set(issue, "closed", "")
		ok = ok && lit.Set(issue, "updated", stamp)
		if !ok {
			log.Fatalln("reopen: Error updating issue fields")
		}
	}
	storeIssues("reopen")
}

func listInfo(id string, issue *dgrl.Branch) string {
	status, _ := lit.Get(issue, "status")
	typ, _ := lit.Get(issue, "type")
	priority, _ := lit.Get(issue, "priority")
	assigned, _ := lit.Get(issue, "assigned")
	summary, _ := lit.Get(issue, "summary")
	return fmt.Sprintf("%s %-7.7s %-7.7s %-7.7s %-7.7s %s",
		id, status, typ, priority, assigned, summary)
}

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
