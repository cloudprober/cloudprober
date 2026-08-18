# How to Contribute

We'd love to accept your contributions to this project. There are just a few
small guidelines you need to follow.

* All submissions, including submissions by project members, require review. We
  use GitHub pull requests for this purpose. Consult [GitHub
  Help](https://help.github.com/articles/about-pull-requests/) for more
  information on using pull requests.

* Please send a PR only if you use Cloudprober yourself and the change is a
  feature or a bug fix that affects you. Changes that come out of actually
  running Cloudprober are the ones we can review and maintain well.

* Every PR should have an associated issue. If there isn't one already, please
  file an [issue](https://github.com/cloudprober/cloudprober/issues), or start a
  [discussion](https://github.com/cloudprober/cloudprober/discussions) if you're
  not sure yet, so that we can agree on the approach before you spend time on
  the code.

* Please keep in mind that Cloudprober’s priority is not to add new features
  "quickly", but to evolve and grow in a mindful way, keeping the codebase small, cohesive, and easy to reason about.

* Features requested by multiple users are prioritized, for implementation as
  well as review.

* Please keep your PRs small, so that they are easier to review. Large PRs are
  less likely to be reviewed and accepted.

* Please avoid adding new dependencies unless absolutely required by the
  functionality.

* We try to limit comment lines to 80 chars. Having comment lines that are
  arbitrarily long makes them rather uncomfortable to read.

## Regarding LLM Generated Code

I use coding assistants all the time, but I don't think purely LLM generated
code works well for an already established software like Cloudprober. It
tends to miss the nuance required to keep the codebase consistent, and
increases the review cost dramatically.