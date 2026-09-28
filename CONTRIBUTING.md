# How to Contribute

## Writing code is cheap now

With coding agents, anyone can produce a plausible-looking PR in minutes.
Writing the code is no longer the hard part; reviewing it, understanding it,
and maintaining it for years is. That cost falls on the maintainers, so we
have a high bar for accepting PRs.

**The most important bar: your change should come from something you ran into
while running Cloudprober in production.** A bug that bit you, a limitation
that blocked you, a feature your setup actually needs. Tell us about it in the
issue: what you were running, what happened, and why it matters to you.

What we'll generally close without a detailed review:

* Flyby PRs that come from wanting to contribute rather than from a real
  need. Everybody wants to contribute now, and we can't review all of it.
* PRs that come out of automated or AI-driven code scanning, or that fix
  edge cases no one has actually run into.
* General "improvements": refactors, cleanups, style changes, or extra tests
  that don't come from a concrete problem.

If you don't use Cloudprober yourself, the most useful thing you can do is
not send a PR. If you do, and something is broken or missing, we want to
hear about it.

## Guidelines

* Every PR should have an associated issue. If there isn't one already, please
  file an [issue](https://github.com/cloudprober/cloudprober/issues), or start a
  [discussion](https://github.com/cloudprober/cloudprober/discussions) if you're
  not sure yet, so that we can agree on the approach before any code is
  written. Often the issue alone is the most valuable contribution.

* All submissions require review. We use GitHub pull requests for this purpose.
  Consult [GitHub Help](
  https://help.github.com/articles/about-pull-requests/) for more information on
  using pull requests.

* Cloudprober's priority is not to add new features "quickly", but to evolve
  and grow in a mindful way, keeping the codebase small, cohesive, and easy to
  reason about.

* Features requested by multiple users are prioritized, for implementation as
  well as review.

* Please keep your PRs small, so that they are easier to review. Large PRs are
  less likely to be reviewed and accepted.

* Please avoid adding new dependencies unless absolutely required by the
  functionality.

* We try to limit comment lines to 80 chars. Having comment lines that are
  arbitrarily long makes them rather uncomfortable to read.

## Using coding agents

I use coding agents all the time, and you're welcome to use them too. But you
own the change: you should understand every line of it, be able to explain
why it's written the way it is, and have tested it in your own setup.

Purely agent-generated code tends to miss the nuance required to keep an
established codebase like Cloudprober consistent, and it increases the review
cost dramatically. If a PR reads like it was generated and not reviewed by
its author, we'll close it.
