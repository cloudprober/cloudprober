---
title: "About"
lead: ""
draft: false
images: []
weight: 300
---

## Origin

I started working on Cloudprober in 2016. I was leading the Cloud Networking SRE
team at Google at that time. As Google Cloud started to grow big time, we
started running into observability issues all over. Customers were finding
problems before us, resulting in a lot of time being spent in troubleshooting
and communicating back and forth[^1].

[^1]:
    A customer-reported infrastructure issue is much harder to debug than an
    issue discovered by your own observability stack.

Problem was that Google's existing observability tools didn't work for the
external Cloud. We needed to build things from ground up. As probers[^2] are
pretty much the foundation of the reliable observability at Google, we decided
to prioritize the development of a prober for Cloud. That's how the journey of
Cloudprober began.

[^2]:
    Almost all of Google's systems rely on probers to detect customer facing
    problems.

Even though the primary goal of Cloudprober at that time was to discover and
alert on Cloud Networking availability and performance problems, we decided to
develop it as a generic prober that could be used to monitor a wide variety of
systems and services. We also decided to make Cloudprober open source so that a
wider community could trust it, contribute to it, and run it on their systems.

Keeping Cloudprober generic and ready for open-source paid off. More and more
Cloud teams started using it internally, and our open-source readiness
commitment made sure we kept our interfaces clean.

## Built for Scale

Cloudprober was built to probe 100s of 1000s of endponts (IPs and HTTP URLs),
while keeping resources, and more importantly, management overhead very very
low. That's the reason Cloudprober tries to be frgual with resources, maximizes
resources utilization relying heavily on Go concurrency, supports probing large
number of targets in parallel at a high freuency (in milliseconds), minimizes
the need of frequent rollouts by supporting dynamic targets discovery, has
native implementations for common probe types, and so on.

## Beyond Google and Open-Source

During early days, Cloudprober's primary focus was being scalable, reliable, and
easy to manage. Things began to change as more and more Cloud products started
using it, and users started asking for more features. However, the real shift to
features happened after we open-sourced Cloudprober in 2017.

We added multitude of other features over time. A few big additions were first
class Kubernetes support, PostgreSQL and Cloudwatch surfacers, OAuth support,
validators. We used the same code base for the internal and the open-source
version, which created a good ecosystem in a way -- we had a large deployment
internally which provided a continuous testing platform for the scalability and
performance aspect of Cloudprober while it was going through some big changes.

## Move away from Google Github

I left Google in Nov 2021. To keep working on Cloudprober independently, I moved
Cloudprober's Github repository from
<a href="https://github.com/google/cloudprober">github.com/google/cloudprober<a>
to
<a href="https://github.com/cloudprober/cloudprober">github.com/cloudprober/cloudprober</a>.
It was a disruptive move (and we lost a lot of Github stars in the process
:smiley:), but it had to be done one day for Cloudprober to become an indepedent
entity and grow even faster. Google still uses Cloudprober, likely even more
widely now, based on the interactions with the Googlers.

## Growth and stability

Cloudprober has continued to evolve and grow over time, but Cloudprober's growth
has been structured. We've made a great effort to control the entropy and making
sure that it doesn't become hard to maintain and grow over time.
