package main

import (
	"fmt"
	"log"
	"os"
	"strings"

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
flit edit [<id>]       Edit issue (default: current)
flit attach <file(s)>  Attach file to current issue`

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
	// case "attach":
	// attachCmd()
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
	if err != nil {
		log.Fatalf("new: %s\n", err)
	}
	err = it.AppendIssues()
	if err != nil {
		log.Fatalf("new: %s\n", err)
	}
	fmt.Println(id)
}

func stateCmd() {
	// verifyRepo()
	// for _, id := range specIds(args) {
	// fmt.Println(stateSummary(id))
	// }
}

func idCmd() {
	loadIssues()
	for _, id := range specIds(args) {
		if it.Issue(id) != nil {
			fmt.Println(id)
		}
	}
}

func showCmd() {
	loadIssues()
	for _, id := range specIds(args) {
		fmt.Println(it.Issue(id))
	}
}

func setCmd() {
	// verifyRepo()
	// if len(args) < 2 {
	// log.Fatalln("set: You must specify a key and value")
	// }
	// if !it.SetWorkingValue(args[0], args[1]) {
	// log.Fatalln("set: Error setting value")
	// }
}

func editCmd() {
	// editor := os.Getenv("VISUAL")
	// if editor == "" {
	// editor = os.Getenv("EDITOR")
	// }
	// if editor == "" {
	// log.Fatalln("VISUAL or EDITOR environment variable must be set")
	// }
	// verifyRepo()
	// id := it.CurrentIssue()
	// isCur := true
	// if len(args) > 0 {
	// id = gitit.FormatId(args[0])
	// isCur = false
	// }
	// verifyIssue(id)
	// if !isCur {
	// err := it.OpenIssue(id)
	// if err != nil {
	// log.Fatalln("edit: Unable to open issue " + id)
	// }
	// }
	// cmd := exec.Command(editor, it.IssueFilename())
	// cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	// err := cmd.Run()
	// if err != nil {
	// log.Fatalln(err)
	// }
}

func attachCmd() {
	// if len(args) == 0 {
	// log.Fatalln("attach: You must specify a file to attach")
	// }
	// verifyRepo()
	// for i := range args {
	// if it.AttachFile(args[i]) != nil {
	// log.Fatalln("attach: Error attaching " + args[i])
	// }
	// }
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

func loadIssues() {
	err := it.Load()
	if err != nil {
		log.Fatalf("show: %s\n", err)
	}
}
