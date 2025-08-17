# Agents

## Git Commits

For git commit messsages, be sure to write them in a format that follows the Linux kernel commit conventions. For example, break at 72 characters, 50 line title and so on. Be sure to include the context of the change, including:

* Why you made the change
* What design notes are useful for future engineers. For any changes that modify public APIs (both in and out of process), be sure to note that there is a breaking change in this commit message by following the syntax:

BREAKING CHANGE: <short-summary-of-api>
  <steps-to-migrate-between-the-old-and-new-api>

Commit meessages should generally be of the form:

```
This is the title of the commit message

This is the body of the commit messsage, in which you
will describe the different things that you changed, and
why you changed them.

== Design Notes
=== Specific Callout

This is a specific callout for a design choice that you
why you made it, what alternatives existed and why this
is superior.
```
