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

lit [help | usage]              Show usage
lit init                        Initialize new issue tracker
lit new                         Create new issue
lit list [<sort>] <spec>        Show summary list of issues in spec
lit id [<sort>] <spec>          List ids in spec
lit show [<sort>] <spec>        Show issues in spec
lit set <field> <val> <spec>    Set issue field
lit comment <id> [<text>]       Add issue comment (opens editor if no text given)
lit edit <spec>                 Edit issues in spec
lit close <spec>                Close issues in spec
lit reopen <spec>               Reopen closed issues in spec

sort: (sortby|rsortby) <field>  Sort (reverse if rsortby) based on field

spec: all | <ids> | (with|without) <field> [<val>] | (less|greater) <field> <val>
      If field is 'comment', compare contents or timestamps based on search type`

// id, closed?, priority, assigned, tags, summary
const listFmt = "%-8.8s %-1.1s %-1.1s %-8.8s %-17.17s %s"

var (
	args    = os.Args[1:]
	it      = lit.New()
	listHdr = fmt.Sprintf(listFmt, "id", "c", "p", "assigned", "tags", "summary")
	cmd     string
)

func main() {
	log.SetFlags(0)
	log.SetPrefix("lit: ")

	// append args piped in from stdin
	if stat, err := os.Stdin.Stat(); err == nil && stat.Mode()&os.ModeNamedPipe != 0 {
		if stdin, err := ioutil.ReadAll(os.Stdin); err == nil {
			stdinArgs := strings.Fields(string(stdin))
			args = append(args, stdinArgs...)
		}
	}

	if len(args) > 0 {
		cmd = args[0]
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
	case "comment":
		commentCmd()
	case "edit":
		editCmd()
	case "close", "reopen":
		closeCmd()
	default:
		log.Fatalln(cmd + " is not a valid command")
	}
}

func usageCmd() {
	fmt.Println(usage)
}

func initCmd() {
	err := it.Init()
	checkErr(err)
}

func newCmd() {
	loadIssues()
	id, err := it.NewIssue()
	checkErr(err)
	storeIssues()
	fmt.Println(id)
}

func listCmd() {
	loadIssues()
	doSort, field, doAscend := dispOpts()
	ids := specIds()
	if doSort {
		it.Sort(ids, field, doAscend)
	}
	fmt.Println(listHdr)
	for _, id := range ids {
		issue := it.Issue(id)
		if issue != nil {
			fmt.Println(listInfo(id, issue))
		}
	}
}

func idCmd() {
	loadIssues()
	doSort, field, doAscend := dispOpts()
	ids := specIds()
	if doSort {
		it.Sort(ids, field, doAscend)
	}
	for _, id := range ids {
		if it.Issue(id) != nil {
			fmt.Println(id)
		}
	}
}

func showCmd() {
	loadIssues()
	doSort, field, doAscend := dispOpts()
	ids := specIds()
	if doSort {
		it.Sort(ids, field, doAscend)
	}
	for _, id := range ids {
		fmt.Println(it.Issue(id))
	}
}

func setCmd() {
	if len(args) < 3 {
		log.Fatalln("set: you must specify a field, value, and spec")
	}
	key, val := args[0], args[1]
	args = args[2:]
	loadIssues()
	stamp := lit.Stamp()
	for _, id := range specIds() {
		issue := it.Issue(id)
		if issue == nil {
			log.Printf("set: error finding issue %s\n", id)
			continue
		}
		ok := lit.Set(issue, key, val)
		ok = ok && lit.Set(issue, "updated", stamp)
		if !ok {
			log.Printf("set: error updating fields in issue %s\n", id)
			continue
		}
	}
	storeIssues()
}

func editCmd() {
	editor := getEditor()
	if editor == "" {
		log.Fatalln("edit: VISUAL or EDITOR environment variable must be set")
	}
	if len(args) < 1 {
		log.Fatalln("edit: you must specify a spec to edit")
	}

	loadIssues()

	// create temp file
	tempFile, err := ioutil.TempFile("", "lit-")
	checkErr(err)
	filename := tempFile.Name()

	// load issue content into temp file
	ids := specIds()
	for _, id := range ids {
		issue := it.Issue(id)
		if issue == nil {
			log.Printf("edit: error finding issue %s\n", id)
			continue
		}
		if !lit.Set(issue, "updated", lit.Stamp()) {
			log.Printf("edit: error setting update time for issue %s\n", id)
			continue
		}
		fmt.Fprintln(tempFile, issue)
	}
	tempFile.Close()

	// get original file state
	origStat, err := os.Stat(filename)
	checkErr(err)

	// launch editor
	ed := exec.Command(editor, filename)
	ed.Stdin, ed.Stdout, ed.Stderr = os.Stdin, os.Stdout, os.Stderr
	err = ed.Run()
	checkErr(err)

	// get updated file state, compare to original
	newStat, err := os.Stat(filename)
	checkErr(err)
	if newStat.ModTime() == origStat.ModTime() {
		log.Fatalln("edit: file unchanged")
	}

	// parse issue from temp file
	tempFile, err = os.Open(filename)
	checkErr(err)
	edited := dgrl.NewParser().Parse(tempFile)
	tempFile.Close()
	if edited == nil {
		log.Fatalln("edit: error parsing file")
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
			if node.Type() == dgrl.BranchType && strings.HasPrefix(node.Key(), id) {
				if editedIssue, ok := node.(*dgrl.Branch); ok {
					*issue = *editedIssue
					didUpdate = true
					break
				}
			}
		}
	}
	if !didUpdate {
		log.Fatalln("edit: did not update anything")
	}

	storeIssues()
}

func closeCmd() {
	if len(args) < 1 {
		log.Fatalf("%s: you must specify a spec\n", cmd)
	}
	loadIssues()
	stamp := lit.Stamp()
	for _, id := range specIds() {
		issue := it.Issue(id)
		if issue == nil {
			log.Printf("%s: error finding issue %s\n", cmd, id)
			continue
		}
		closedStamp := ""
		if cmd == "close" {
			closedStamp = stamp
		}
		ok := lit.Set(issue, "closed", closedStamp)
		ok = ok && lit.Set(issue, "updated", stamp)
		if !ok {
			log.Printf("%s: error updating fields for issue %s\n", cmd, id)
			continue
		}
	}
	storeIssues()
}

func commentCmd() {
	if len(args) < 1 {
		log.Fatalln("comment: you must specify an issue to comment on")
	}
	id := args[0]
	loadIssues()
	issue := it.Issue(id)
	if issue == nil {
		log.Fatalf("comment: error finding issue %s\n", id)
	}
	comment := ""
	if len(args) > 1 {
		comment = args[1]
	} else {
		comment = editComment()
	}
	stamp := lit.Stamp()
	commentBranch := dgrl.NewBranch(stamp)
	commentBranch.Append(dgrl.NewText(comment))
	issue.Append(commentBranch)
	if !lit.Set(issue, "updated", stamp) {
		log.Printf("comment: error setting update time for issue %s\n", id)
	}
	storeIssues()
}

func editComment() string {
	editor := getEditor()
	if editor == "" {
		log.Fatalln("comment: VISUAL or EDITOR environment variable must be set")
	}
	// create temp file
	tempFile, err := ioutil.TempFile("", "lit-")
	checkErr(err)
	filename := tempFile.Name()

	// get original file state
	origStat, err := os.Stat(filename)
	checkErr(err)

	// launch editor
	ed := exec.Command(editor, filename)
	ed.Stdin, ed.Stdout, ed.Stderr = os.Stdin, os.Stdout, os.Stderr
	err = ed.Run()
	checkErr(err)

	// get updated file state, compare to original
	newStat, err := os.Stat(filename)
	checkErr(err)
	if newStat.ModTime() == origStat.ModTime() {
		log.Fatalln("comment: file unchanged")
	}

	// read comment from file
	commentData, err := ioutil.ReadFile(filename)
	checkErr(err)
	return string(commentData)
}

func listInfo(id string, issue *dgrl.Branch) string {
	status := " "
	closed, _ := lit.Get(issue, "closed")
	if len(closed) > 0 {
		status = "*"
	}
	tags, _ := lit.Get(issue, "tags")
	priority, _ := lit.Get(issue, "priority")
	assigned, _ := lit.Get(issue, "assigned")
	summary, _ := lit.Get(issue, "summary")
	return fmt.Sprintf(listFmt, id, status, priority, assigned, tags, summary)
}

func keyval(kv []string) (string, string) {
	key, val := "", ""
	if len(kv) > 0 {
		key = kv[0]
	}
	if len(kv) > 1 {
		val = kv[1]
	}
	return key, val
}

func matchIds(kv []string, doesMatch bool) []string {
	key, val := keyval(kv)
	return it.Match(key, val, doesMatch)
}

func compareIds(kv []string, isLess bool) []string {
	key, val := keyval(kv)
	return it.Compare(key, val, isLess)
}

func dispOpts() (bool, string, bool) {
	switch {
	case len(args) == 0:
		return false, "", true
	case args[0] == "sortby" || args[0] == "rsortby":
		if len(args) < 2 {
			log.Fatalf("%s: sort requested, but no field given to sort by\n", cmd)
		}
		doSort := true
		doAscend := (args[0] == "sortby")
		field := args[1]
		args = args[2:]
		return doSort, field, doAscend
	}
	return false, "", true
}

func specIds() []string {
	ids := []string{}
	switch {
	case len(args) == 0:
		return ids
	case args[0] == "with":
		ids = matchIds(args[1:], true)
	case args[0] == "without":
		ids = matchIds(args[1:], false)
	case args[0] == "less":
		ids = compareIds(args[1:], true)
	case args[0] == "greater":
		ids = compareIds(args[1:], false)
	case args[0] == "all":
		ids = it.IssueIds()
	default:
		ids = args
	}
	return ids
}

func loadIssues() {
	err := it.Load()
	checkErr(err)
}

func storeIssues() {
	err := it.Store()
	checkErr(err)
}

func checkErr(err error) {
	if err != nil {
		log.Fatalf("%s: %s\n", cmd, err)
	}
}

func getEditor() string {
	editor := os.Getenv("VISUAL")
	if editor == "" {
		editor = os.Getenv("EDITOR")
	}
	return editor
}
