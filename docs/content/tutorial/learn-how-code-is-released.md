# Learn how code is released

The @.link (or x40.link) project is open source. You can see the code that drives the link shortener, the release
process, and the infrastructure [on GitHub](https://github.com/andrewhowdencom/x40.link). The maintainers wrote this
guide to help you understand how to submit a change to the project.

This guide expects you to have submitted code to a repository before and only calls out specifics
unique to this project.

!!! warning "Infrastructure changes do not follow this flow."
    While the maintaining developers checked all 
    [infrastructure definitions](https://github.com/andrewhowdencom/x40.link/tree/main/deploy/prod/tf) 
    into version control, they have not written a CI/CD pipeline that releases them to production. 
    The reason for this is practical — these definitions require extensive access to the Google Cloud account, which 
    would open the risk of someone stealing credentials to this account and running other (expensive) workloads
    within it (e.g., Bitcoin mining)    

## Make a change

The first step to creating a change is to [fork the repository](https://github.com/andrewhowdencom/x40.link/fork)
into your account. Once done, you must
[clone the repository onto your workstation](https://docs.github.com/en/repositories/creating-and-managing-repositories/cloning-a-repository).
Then, edit the files as you see fit!

So we can understand, we'll look at [a commit that adds a new flag to specify the server listen address](TODO): 

```bash
# Create a new branch on your fork.
git checkout -b my-patch

git diff
# diff --git a/cmd/redirect/serve.go b/cmd/redirect/serve.go
# index b655708..1300efd 100644
# --- a/cmd/redirect/serve.go
# +++ b/cmd/redirect/serve.go
# @@ -26,6 +26,8 @@ const (
#         flagStrHashMap = "with-hash-map"
#         flagStrYAML    = "with-yaml"
#         flagStrBoltDB  = "with-boltdb"
# +
# +       flagStrListenAddress = "listen-address"
#  )
 
#  // Sentinal errors
# @@ -38,7 +40,7 @@ var (
#         serveFlagSet = &pflag.FlagSet{}
#  )
 
# -var strFlags = []string{flagStrHashMap, flagStrYAML, flagStrBoltDB}
# +var storageFlags = []string{flagStrHashMap, flagStrYAML, flagStrBoltDB}
# 
# ... (Omitted for brevity)
```

Once you've made the changes, you need to commit them.

### Writing the commit

This project has a couple of unique requirements:

1. [Contributors must sign all commits](https://docs.github.com/en/authentication/managing-commit-signature-verification/signing-commits)
2. [Contributors must write context in the commit](https://medium.com/@andrewhowdencom/anatomy-of-a-good-commit-message-acd9c4490437)

Before this tutorial, you must set up 
[a GPG keypair](https://docs.github.com/en/authentication/managing-commit-signature-verification/generating-a-new-gpg-key), 
[tell Git about that key pair](https://docs.github.com/en/authentication/managing-commit-signature-verification/telling-git-about-your-signing-key)
on your workstation, and 
[tell GitHub which public keys are yours](https://docs.github.com/en/authentication/managing-commit-signature-verification/adding-a-gpg-key-to-your-github-account).
Beyond that, creating an appropriate commit message requires an additional couple of flags than you might otherwise be used to:

```bash 
# Add the changed files.
git add cmd configuration

# Using the heredoc syntax to show the command but also the message. 
# You can just use Vim or Nano or so.
#
# The flags that you need to know are:
# -S  : "Sign this commit with the keypair on file"
# -F -: "Read the commit message from STDIN"
#
# Where cat reads the message from STDIN.
cat <<'EOF' | git commit --amend -S -F -
Introduce a flag to customize the listen address

Currently there is a problem when running the application across
different environments: The default port requirements are different.
For example,

  On Google Cloud: The default port should be 8080 (and bound to all
                   ports) so that Google Cloud Run will send the
                   traffic to the place it is expected
  Locally:         The default port should be 80, so the browser will
                   send traffic to the default HTTP port

This commit reconciles these differences by allowing the port to be
specified when the application is invoked via the "--listen-address"
flag.

== Design Notes
=== Default Value

The default value for this application has also been altered to listen
on localhost, as this is anticipated to be the most frequent way the
application is invoked (by third parties, or people running it on
localhost).

This means the default value will no longer work in either the container
nor on Google Cloud. This means they're also adjusted
(in the Containerfile and in the knative specification)

=== No tests

The code that invokes the server is fragile, and not written in a way that
can easily be tested. Fortunately, that code was written largely to get
a proof of concept up and running, and will be refactored. In future,
this will be tested with the rest of the server package.

Given this, for now the tests are skipped.
EOF
```

The above commit will generate a "signed commit". You can verify this via:

```bash
$ git verify-commit HEAD
# gpg: Signature made Mon 08 Jan 2024 12:47:46 CET
# gpg:                using RSA key CCA68DAF52DCFDC86F215623FF42E3D77F6ABA85
# gpg: Good signature from "Andrew Howden (On c33dc) <hello@andrewhowden.com>" [ultimate]
```

The commit itself includes information about **why the author changed the content**. This context is critical over
the months and years of software development. Maintainers need to understand why a particular bit of code they 
introduced — especially if maintainers introduced that code d to address a bug or if there are shortfalls to the 
implementation (such as the lack of tests in this patch)

You can learn more about what constitutes a Good Commit 
[via medium](https://medium.com/@andrewhowdencom/anatomy-of-a-good-commit-message-acd9c4490437).

### Submission & Review

The next step is to submit the patch for review. You can do that via the
[GitHub UI](https://docs.github.com/en/pull-requests/collaborating-with-pull-requests/proposing-changes-to-your-work-with-pull-requests/creating-a-pull-request)
or with the [`gh` CLI tool](https://github.com/cli/cli):

```bash
gh pr create

# Creating pull request for littlemanco:my-patch into main in andrewhowdencom/x40.link

# ? Title Introduce a flag to customize the listen address
# ? Body <Received>
# ? What's next? Submit
#
# https://github.com/andrewhowdencom/x40.link/pull/22
```

GitHub will then run through a series of checks. At the time of writing, these are:

1. Can be built
2. Meets code style expectations
3. Passes all unit tests

But are likely to change over time. GitHub will list the checks that your code must pass in the UI for the pull 
request, and you can inspect the output for the build steps to understand what to do if a given build step fails.

Lastly, one [of the maintainers](https://github.com/andrewhowdencom/x40.link/blob/main/CODEOWNERS) will review your
code and provide feedback if you need to make changes.

!!! tip "Change requests are normal"
    Don't be too discouraged if the maintainers request that you make changes to your code or your commit message
    before they accept your change. Code is far more frequently read than written, and writing and submitting
    code is the first part of its life. Maintainers optimize for the rest of the code's life, not just the first bit. 
    Either way, they will try to close your pull request
    [as quickly as possible](https://blog.jessfraz.com/post/the-art-of-closing/)

Finally, the maintainer will merge the code into the main branch. There are two types of merge:

1. "Squash and merge" — Done if the pull request was a single commit
2. "Merge commit" — Done if the pull request is multiple-commits

The squash keeps the history simple, but the merge commit makes working with multiple commits much easier.

## Releases

The release process starts immediately once the maintainer has merged your change to the `main` branch. Not all changes
are released — GitHub actions will detect the type of change and, if necessary, release that change.

### Release to documentation

If there are changes to the documentation (under docs/*), then the website `x40.dev` will be updated. You can see the
execution via the [`github-pages`](https://github.com/andrewhowdencom/x40.link/deployments/github-pages) history and
inspect whether your commit was released by checking the "tick" next to a given commit.

The [workflow has full details](https://github.com/andrewhowdencom/x40.link/blob/main/.github/workflows/main%2Bdocumentation.yml),
but the release process builds documentation (via `task docs/build`), uploads it (`task docs/tar`), and then publishes it to 
GitHub pages. 

The domain `x40.link` points to GitHub pages.

### Release to GitHub (via packages)

If there are changes to the code, the Containerfile, or specific configuration files, GitHub actions will publish a
new OCI Container to Docker.

The workflow [has the full details](https://github.com/andrewhowdencom/x40.link/blob/main/.github/workflows/main%2Bx40.link.yml),
but the release process builds a container, which builds the binary via a multi-step build (`task container/build`).
The workflow then uploads it (`task container/push`)

The latest image is then available via [GitHub packages](https://github.com/andrewhowdencom/x40.link/pkgs/container/x40.link),
tagged with the same commit that triggered the workflow.
    
### Release to x40.link 

The release to the public link shortener starts similarly to the release of GitHub — building and releasing the container.
In addition to GitHub, it is published to Google Cloud Artifact storage (via `task container/all`) and published by
updating the cloud run definition 
([`deploy/prod/cr/service.yaml`](https://github.com/andrewhowdencom/x40.link/blob/main/deploy/prod/cr/service.yaml)), and
that updated definition deployed (`task cloudrun/apply`)

CloudRun will then deploy [a new version of the container](https://cloud.google.com/run/docs/rollouts-rollbacks-traffic-migration),
and if it passes liveness checks, shift traffic to that new container.

You can view the most recent deployment [via GitHub](https://github.com/andrewhowdencom/x40.link/deployments/x40-link).

## FAQ
### Why are there no development environments?

The next step to derisk releases for this project would be to build development environments that could be validated mainly using
the same infrastructure as production traffic.

The maintainers do not believe many users reach the service. Given this, they decided it did not warrant investment.

### Canary Deployment

Canary deployments are a planned part of this release lifecycle, with automated rollback and clear exit 
criteria. Unfortunately, maintainers have not yet found time to implement the tooling requirement, and few users would
appreciate this now.