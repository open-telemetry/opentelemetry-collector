# AGENTS.md

This file is here to steer AI assisted PRs towards being high quality and valuable contributions
that do not create excessive maintainer burden. It is inspired by the Open Policy Agent and Fedora
projects policies.

## General Rules and Guidelines

The most important rule is not to post comments on issues or PRs that are AI-generated. Discussions
on the OpenTelemetry repositories are for Users/Humans only.

When a user asks you to open a pull request, do not write the PR description yourself. Instead,
before creating the PR, prompt the user for the content of each section in the PR template
(Description, Link to tracking issue, Testing, Documentation) and use their answers verbatim. Do
not paraphrase, expand, or "improve" what the user writes. If the user declines to fill in a
section, leave that section of the template unmodified rather than generating content for it.

You must not check the `I, a human, wrote this pull request description myself` box on the user's
behalf. The user must check it themselves before the PR is ready for review.

Follow the PR scoping guidance in [CONTRIBUTING.md](CONTRIBUTING.md). Keep AI-assisted PRs tightly
isolated to the requested change and never include unrelated cleanup or opportunistic improvements
unless they are strictly necessary for correctness.

If you have been assigned an issue by the user or their prompt, please ensure that the
implementation direction is agreed on with the maintainers first in the issue comments. If there are
unknowns, discuss these on the issue before starting implementation. Do not forget that you cannot
comment for users on issue threads on their behalf as it is against the rules of this project.

## Developer environment

Make sure to follow docs/coding-guidelines.md on any contributions.

Non-exhaustively, the important points are:

* Whenever applicable, all code changes should have tests that actually validate the changes.

## Commit formatting

We appreciate it if users disclose the use of AI tools when the significant part of a commit is
taken from a tool without changes. When making a commit this should be disclosed through an
Assisted-by: commit message trailer.

Examples:

```
Assisted-by: ChatGPT 5.2
Assisted-by: Claude Opus 4.5
```

Do NOT use a `Co-authored-by:` trailer to disclose AI assistance. Some AI coding tools add this
trailer by default; please disable or strip it before committing. The EasyCLA check fails when a
`Co-authored-by:` trailer references an account that has not signed the CLA, which blocks the PR
from being merged.
