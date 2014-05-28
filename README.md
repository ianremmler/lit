lit
===

Lit is a lightweight issue tracker.

To install lit, install Go if it isn't already, and run:

```
go get github.com/ianremmler/lit/cmd/lit
```

Run `lit` with no arguments to see how to use it.

Issues are stored in a single text file in
[Doggerel](https://github.com/ianremmler/dgrl) format.

Some [scripts](https://github.com/ianremmler/lit/tree/master/scripts)
are included to enable complex queries from the command line.  For
example, to list the issues that are open and either assigned to user
bob or have priority greater than 1:

```
((lit id with assigned bob; lit id greater priority 1) | lit-or; \
 lit id without closed) | lit-and | xargs lit list
```
