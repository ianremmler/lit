lit
===

Lit is a lightweight issue tracker.

To install lit, install Go if it isn't already, and run:

```
go get github.com/ianremmler/lit/cmd/lit
```

Run `lit help` to see how to use it.

Currently, the only configuration available is the environment variable
`LIT_USER` which, if set, will be used instead of the current username.

Issues are stored in a single text file in
[Doggerel](https://github.com/ianremmler/dgrl) format.

Some [scripts](https://github.com/ianremmler/lit/tree/master/scripts) are
included to enable complex queries from the command line.  Using shell command
substitution, the syntax looks a bit like Lisp.  For example, to list the
issues that are open and either assigned to user bob or have priority greater
than 1:

```
# Note: 'lit id' returns ids of open issues
lit list $(andl $(lit id) \
                $(orl $(lit with assigned bob) \
                      $(lit greater priority 1)))
```
