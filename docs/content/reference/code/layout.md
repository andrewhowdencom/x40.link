# Layout

The [project](https://github.com/andrewhowdencom/x40.link) contains all files related to the codebase, to building the
code, defining the infrastructure, deploying it to production, or anything else that might need to be delivered. It is
arranged in a series of subfolders, including:

| Directory     | Description                                                                                             |
|:--------------|:--------------------------------------------------------------------------------------------------------|
| cmd           | Command line interface                                                                                  |
| configuration | Configuration paths and utility functions                                                               |
| deploy        | Infrastructure as code definitions                                                                      |
| dist          | Generated artifacts (e.g. binaries, tarballs)                                                           |
| docs          | Documentation, including the mkdocs installation                                                        |
| storage       | Pluggable storage interface for where to lookup URLs                                                    |