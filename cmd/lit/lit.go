package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"os/user"
	"strconv"
	"strings"

	"github.com/ianremmler/dgrl"
	"github.com/ianremmler/lit"
)

const usage = `lit help                        Display usage information
lit init                        Initialize new issue tracker
lit new [<num>]                 Create num new issues (default: 1)
lit [id] [<sort>] <spec>        Show ids of specified issues
lit list [<sort>] <spec>        List specified issues
lit show [<sort>] <spec>        Show specified issues
lit set <key> <val> <spec>      Set value for key in specified issues
lit tag (add|del) <tag> <spec>  Add or delete tag in specified issues
lit comment <id> [<text>]       Add issue comment (default: edit text)
lit edit <spec>                 Edit specified issues
lit close <spec>                Close specified issues
lit reopen <spec>               Reopen specified issues
lit attach (add <id> <file> [<desc>] | show <id> <file> | list <id>)
	Add, show, or list issue attachments

sort: (sortby|rsortby) <key>
	Sort (reverse if rsortby) based on key

spec: open | closed | all | <ids> |
      (with | without | less | greater) <key> [<val>]
	Specifies which issues to operate on
	Use 'comment' key to filter by comment contents and times
	Use 'attach' key to filter by attachment names and counts`

const (
	// id, closed?, priority, attached, assigned, tags, summary
	listFmt = "%-8.8s %-1.1s %-1.1s %-1.1s %-8.8s %-15.15s %s"
)

var (
	args     = os.Args[1:]
	it       = lit.New()
	listHdr  = fmt.Sprintf(listFmt, "id", "c", "p", "a", "assigned", "tags", "summary")
	username = "?"
	cmd      = "id"
)

func main() {
	log.SetFlags(0)
	log.SetPrefix("lit: ")

	if userEnv := os.Getenv("LIT_USER"); userEnv != "" {
		username = userEnv
	} else {
		if user, err := user.Current(); err == nil {
			if host, err := os.Hostname(); err == nil {
				username = fmt.Sprintf("%s@%s", user.Username, host)
			}
		}
	}

	// append args piped in from stdin
	if stat, err := os.Stdin.Stat(); err == nil && stat.Mode()&os.ModeNamedPipe != 0 {
		if stdin, err := ioutil.ReadAll(os.Stdin); err == nil {
			args = append(args, strings.Fields(string(stdin))...)
		}
	}

	if len(args) > 0 {
		cmd = args[0]
		args = args[1:]
	}
	switch cmd {
	case "-h", "-help", "--help", "help":
		usageCmd()
	case "init":
		initCmd()
	case "new":
		newCmd()
	case "id":
		idCmd()
	case "list":
		listCmd()
	case "show":
		showCmd()
	case "set":
		setCmd()
	case "tag":
		tagCmd()
	case "comment":
		commentCmd()
	case "attach":
		attachCmd()
	case "edit":
		editCmd()
	case "close", "reopen":
		closeCmd()
	default:
		cmd, args = "id", append([]string{cmd}, args...)
		idCmd()
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
	numIssues := 1
	if len(args) > 0 {
		num, err := strconv.ParseUint(args[0], 10, 16)
		checkErr(err)
		numIssues = int(num)
	}
	loadIssues()
	issues := it.NewIssues(username, numIssues)
	for _, issue := range issues {
		fmt.Println(issue.Key())
	}
	storeIssues()
}

func idCmd() {
	loadIssues()
	doSort, key, doAscend := dispOpts()
	ids := specIds()
	if doSort {
		it.Sort(ids, key, doAscend)
	}
	for _, id := range ids {
		if issue := it.Issue(id); issue != nil {
			fmt.Println(issue.Key())
		}
	}
}

func listCmd() {
	loadIssues()
	doSort, key, doAscend := dispOpts()
	ids := specIds()
	if doSort {
		it.Sort(ids, key, doAscend)
	}
	fmt.Println(listHdr)
	for _, id := range ids {
		issue := it.Issue(id)
		if issue != nil {
			fmt.Println(listInfo(issue))
		}
	}
}

func showCmd() {
	loadIssues()
	doSort, key, doAscend := dispOpts()
	ids := specIds()
	if doSort {
		it.Sort(ids, key, doAscend)
	}
	for _, id := range ids {
		issue := it.Issue(id)
		if issue == nil {
			log.Printf("show: error finding issue %s\n", id)
			continue
		}
		fmt.Println(issue)
	}
}

func setCmd() {
	if len(args) < 2 {
		log.Fatalln("set: you must specify a key and value")
	}
	key, val := args[0], args[1]
	args = args[2:]
	loadIssues()
	stamp := lit.Stamp(username)
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

func tagCmd() {
	if len(args) < 2 {
		log.Fatalln("tag: you must specify an operation and tag")
	}
	op, tag := args[0], args[1]
	if op != "add" && op != "del" {
		log.Fatalf("tag: %s is not a valid operation\n", op)
	}
	args = args[2:]
	doAdd := (op == "add")

	loadIssues()
	stamp := lit.Stamp(username)
	for _, id := range specIds() {
		issue := it.Issue(id)
		if issue == nil {
			log.Printf("tag: error finding issue %s\n", id)
			continue
		}
		ok := lit.ModifyTag(issue, tag, doAdd)
		ok = ok && lit.Set(issue, "updated", stamp)
		if !ok {
			log.Printf("tag: error updating fields in issue %s\n", id)
			continue
		}
	}
	storeIssues()
}

func commentCmd() {
	if len(args) < 1 {
		log.Fatalln("comment: you must specify an issue")
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
	stamp := lit.Stamp(username)
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
		log.Fatalf("%s: VISUAL or EDITOR environment variable must be set\n", cmd)
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
		log.Fatalf("%s: file unchanged", cmd)
	}

	// read comment from file
	commentData, err := ioutil.ReadFile(filename)
	checkErr(err)
	return string(commentData)
}

func attachCmd() {
	if len(args) < 1 {
		log.Fatalln("attach: you must specify an operation")
	}
	op := args[0]
	switch op {
	case "add":
		addAttach()
	case "list":
		listAttach()
	case "show":
		showAttach()
	default:
		log.Fatalf("attach: %s is not a valid operation\n", op)
	}
}

func addAttach() {
	if len(args) < 3 {
		log.Fatalln("attach: you must specify an issue and file")
	}
	id := args[1]
	loadIssues()
	issue := it.Issue(id)
	if issue == nil {
		log.Fatalf("attach: error finding issue %s\n", id)
	}

	src := args[2]
	_, err := os.Stat(src)
	checkErr(err)

	comment := ""
	if len(args) > 3 {
		comment += args[3]
	} else {
		comment += editComment()
	}

	stamp, err := it.Attach(issue, src, username, comment)
	checkErr(err)
	if !lit.Set(issue, "updated", stamp) {
		log.Printf("attach: error setting update time for issue %s\n", id)
	}
	storeIssues()
}

func listAttach() {
	if len(args) < 2 {
		log.Fatalln("attach: you must specify an issue")
	}
	id := args[1]
	loadIssues()
	issue := it.Issue(id)
	if issue == nil {
		log.Fatalf("attach: error finding issue %s\n", id)
	}
	for _, filename := range it.Attachments(issue) {
		fmt.Println(filename)
	}
}

func showAttach() {
	if len(args) < 3 {
		log.Fatalln("attach: you must specify an issue and file")
	}
	id := args[1]
	loadIssues()
	issue := it.Issue(id)
	if issue == nil {
		log.Fatalf("attach: error finding issue %s\n", id)
	}
	attachment, err := it.GetAttachment(issue, args[2])
	checkErr(err)
	defer attachment.Close()
	_, err = io.Copy(os.Stdout, attachment)
	checkErr(err)
}

func editCmd() {
	editor := getEditor()
	if editor == "" {
		log.Fatalln("edit: VISUAL or EDITOR environment variable must be set")
	}

	loadIssues()

	// create temp file
	tempFile, err := ioutil.TempFile("", "lit-")
	checkErr(err)
	filename := tempFile.Name()

	// load issue content into temp file
	ids := specIds()
	toEdit := dgrl.NewRoot()
	for _, id := range ids {
		issue := it.Issue(id)
		if issue == nil {
			log.Printf("edit: error finding issue %s\n", id)
			continue
		}
		toEdit.Append(issue)
	}
	err = toEdit.Write(tempFile)
	checkErr(err)
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
	edIssues := dgrl.NewParser().Parse(tempFile)
	tempFile.Close()
	if edIssues == nil {
		log.Fatalln("edit: error parsing file")
	}

	// update issues if we find a match
	didUpdate := false
	stamp := lit.Stamp(username)
	for _, id := range ids {
		issue := it.Issue(id)
		if issue == nil {
			// already printed error, so don't repeat here
			continue
		}
		for _, node := range edIssues.Kids() {
			if ed, ok := node.(*dgrl.Branch); ok && strings.HasPrefix(ed.Key(), id) {
				*issue = *ed
				if !lit.Set(issue, "updated", stamp) {
					log.Printf("edit: error setting update time for issue %s\n", id)
					continue
				}
				didUpdate = true
				break
			}
		}
	}
	if !didUpdate {
		log.Fatalln("edit: did not update anything")
	}

	storeIssues()
}

func closeCmd() {
	loadIssues()
	stamp := lit.Stamp(username)
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

func listInfo(issue *dgrl.Branch) string {
	status := " "
	closed, _ := lit.Get(issue, "closed")
	if len(closed) > 0 {
		status = "*"
	}
	tags, _ := lit.Get(issue, "tags")
	priority, _ := lit.Get(issue, "priority")
	attached := " "
	if numAttach := len(it.Attachments(issue)); numAttach > 0 {
		attached = "*"
		if numAttach < 10 {
			attached = strconv.Itoa(numAttach)
		}
	}
	assigned, _ := lit.Get(issue, "assigned")
	summary, _ := lit.Get(issue, "summary")
	return fmt.Sprintf(listFmt, issue.Key(), status, priority, attached, assigned, tags, summary)
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
			log.Fatalf("%s: sort requested, but no key given to sort by\n", cmd)
		}
		doSort := true
		doAscend := (args[0] == "sortby")
		key := args[1]
		args = args[2:]
		return doSort, key, doAscend
	}
	return false, "", true
}

func specIds() []string {
	ids := []string{}
	filt := ""
	if len(args) > 0 {
		filt = args[0]
	}
	switch filt {
	case "all":
		ids = it.IssueIds()
	case "open":
		ids = matchIds([]string{"closed", ""}, false)
	case "closed":
		ids = matchIds([]string{"closed", ""}, true)
	case "with":
		ids = matchIds(args[1:], true)
	case "without":
		ids = matchIds(args[1:], false)
	case "less":
		ids = compareIds(args[1:], true)
	case "greater":
		ids = compareIds(args[1:], false)
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
		str := ""
		if cmd != "" {
			str += cmd + ": "
		}
		log.Fatalf("%s%s\n", str, err)
	}
}

func getEditor() string {
	editor := os.Getenv("VISUAL")
	if editor == "" {
		editor = os.Getenv("EDITOR")
	}
	return editor
}
