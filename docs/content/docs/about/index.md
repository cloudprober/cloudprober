---
title: "About Cloudprober"
lead: ""
draft: false
images: []
weight: 300
date: 2023-08-20
lastmod: 2023-08-20
contributors: ["Manu Garg"]
---

{{% blog-meta author="Manu Garg" publishDate="Aug 20, 2023" readingTime=3 %}}

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

Cloudprober was built to probe 100s of 1000s of endpoints (IPs and HTTP URLs),
while keeping resources, and more importantly, management overhead very very
low. That's the reason Cloudprober tries to be frugal with the resources,
maximizes resources utilization relying heavily on Go concurrency, supports
probing large number of targets in parallel at a high frequency (in
milliseconds), minimizes the need of frequent rollouts by supporting dynamic
targets discovery, has native implementations for common probe types, and so on.

## Beyond Google and Open-Source

During early days, Cloudprober's primary focus was being scalable, reliable, and
easy to manage. Things began to change as more and more Cloud products started
using it, and users started asking for more features. However, the real shift to
features happened after we open-sourced Cloudprober in 2017.

We added a multitude of features over time. A few big additions were:
first-class Kubernetes support, PostgreSQL and Cloudwatch surfacers, OAuth
support, validators, and probe status UI. We used the same code base for the
internal and the open-source version, which created a good ecosystem in a way --
we had a large deployment internally which provided a continuous testing
platform for Cloudprober, especially for its scalability and performance
aspects, while it was going through all the big changes.

## Move away from Google Github

I left Google in Nov 2021. To keep working on Cloudprober independently, I moved
Cloudprober's Github repository from
<a href="https://github.com/google/cloudprober">github.com/google/cloudprober<a>
to
<a href="https://github.com/cloudprober/cloudprober">github.com/cloudprober/cloudprober</a>.
This was a disruptive move (and we lost a lot of Github stars in the process
:smiley:), but it had to be done one day anyway, in order for Cloudprober to
become an independent entity and grow even faster. I can't say this
authoritatively now, but I believe Google still uses Cloudprober, perhaps even
more widely now, based on Googlers' interactions with the project.

## Growth and stability

Cloudprober has been around for a bit, and all this while it has continued to
change and evolve, to meet more and more users' need. That's an important
quality for any software, otherwise software wither over time. To make sure
Cloudprober stays so, we've gone to great lengths to ensure that Cloudprober's
internal interfaces stay clean, even if that requires some refactoring from time
to time.
