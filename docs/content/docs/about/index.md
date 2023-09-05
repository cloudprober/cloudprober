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

During its early days, our priorities for Cloudprober were scalability,
reliability, and ease of management. Things began to change as more and more
Cloud products started using it, and users started asking for more features.
However, the real shift to features happened after we open-sourced Cloudprober
in 2017.

We added a wealth of features over time, such as first-class Kubernetes support,
a built-in probe status UI, PostgreSQL and Cloudwatch surfacers, OAuth support,
Validators, and most recently, built-in alerting capability. We used the same
codebase for the internal and open-source versions, which created a huge
advantage -- we had an extensive deployment internally which provided a
continuous testing platform for Cloudprober, particularly for its scalability
and performance aspects, while it was going through all the big changes.

## Move away from Google Github

I left Google in Nov 2021. To keep working on Cloudprober independently, I moved
Cloudprober's Github repository from
<a href="https://github.com/google/cloudprober">github.com/google/cloudprober<a>
to
<a href="https://github.com/cloudprober/cloudprober">github.com/cloudprober/cloudprober</a>.
This was a disruptive move (and we lost a lot of Github stars in the process
:smiley:), but it had to be done one day anyway, in order for Cloudprober to
become an independent entity and grow even faster. While I can't say this
authoritatively now, I believe Google still uses Cloudprober, possibly even more
widely now, based on Googlers' interactions with the project.

## Growth and stability

Throughout its journey, Cloudprober has continuously adapted and expanded to
meet the evolving needs of its users -- an essential trait for any software.
Without this attribute, software inevitably withers over time. To ensure
Cloudprober thrives and evolves robustly, we've been very diligent that it grows
in a structured way, a commitment we'll uphold in future as well.
