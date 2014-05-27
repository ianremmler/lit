package main

import (
	"fmt"
	"io/ioutil"
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
lit list <spec>               Show list of issues in spec
lit init                      Initialize new issue tracker
lit new                       Create new issue
lit id <spec>                 List ids in spec
lit show <spec>               Show issues in spec
lit set <field> <val> <spec>  Set issue field
lit edit <spec>               Edit issues in spec
lit close <spec>              Close issue in spec
lit reopen <spec>             Reopen closed issues in spec`

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
	case "close", "reopen":
		closeCmd(cmd)
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
			log.Printf("set: Error finding issue %s\n", id)
			continue
		}
		ok := lit.Set(issue, key, val)
		ok = ok && lit.Set(issue, "updated", stamp)
		if !ok {
			log.Printf("set: Error updating fields in %s\n", id)
			continue
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
		log.Fatalln("edit: You must specify a spec to edit")
	}

	// create temp file
	issueFile, err := ioutil.TempFile("", "lit-")
	checkErr("edit", err)
	filename := issueFile.Name()

	// load issue content into temp file
	loadIssues("edit")
	ids := specIds(args)
	for _, id := range ids {
		issue := it.Issue(id)
		if issue == nil {
			log.Printf("edit: Error finding issue %s\n")
			continue
		}
		if !lit.Set(issue, "updated", lit.Stamp()) {
			log.Printf("edit: Error setting update time for %s\n")
			continue
		}
		fmt.Fprintln(issueFile, issue)
	}
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
		log.Fatalln("edit: File unchanged")
	}

	// parse issue from temp file
	issueFile, err = os.Open(filename)
	checkErr("edit", err)
	defer issueFile.Close()
	edited := dgrl.NewParser().Parse(issueFile)
	if edited == nil {
		log.Fatalln("edit: Error parsing file")
	}

	// update issues if we find a match
	didUpdate := false
	for _, id := range ids {
		issue := it.Issue(id)
		if issue == nil {
			// already printed error, so don't repeat here
			continue
		}
		for _, node := range edited.Kids() {
			if node.Type() == dgrl.BranchType && node.Key() == id {
				if editedIssue, ok := node.(*dgrl.Branch); ok {
					*issue = *editedIssue
					didUpdate = true
					break
				}
			}
		}
	}
	if !didUpdate {
		log.Fatalln("edit: Did not update anything")
	}

	storeIssues("edit")
}

func closeCmd(cmd string) {
	if len(args) < 1 {
		log.Fatalf("%s: You must specify a spec\n", cmd)
	}
	loadIssues(cmd)
	stamp := lit.Stamp()
	for _, id := range specIds(args) {
		issue := it.Issue(id)
		if issue == nil {
			log.Printf("%s: Error finding issue %s\n", cmd, id)
			continue
		}
		status := "closed"
		closedStamp := stamp
		if cmd == "reopen" {
			status = "open"
			closedStamp = ""
		}
		ok := lit.Set(issue, "status", status)
		ok = ok && lit.Set(issue, "closed", closedStamp)
		ok = ok && lit.Set(issue, "updated", stamp)
		if !ok {
			log.Printf("%s: Error updating fields for %s\n", cmd, id)
			continue
		}
	}
	storeIssues(cmd)
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
