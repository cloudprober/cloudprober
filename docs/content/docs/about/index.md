---
title: "About Cloudprober"
lead: ""
draft: false
images: []
weight: 300
date: 2023-10-05
lastmod: 2023-10-20
contributors: ["Manu Garg"]
---

{{% blog-meta author="Manu Garg" publishDate="Oct 05, 2023" readingTime="3 min" %}}

## Origin

I started building Cloudprober in 2016, while I was at Google, leading the Cloud
Networking SRE team there. Google Cloud was just beginning to grow big, and we
were still grappling with some early growth issue. Our biggest problem was that
our customers were discovering problems before us, which resulted in bad
experience for our customers and huge time sink for my team in debugging those
issues.[^1].

[^1]:
    A customer-reported infrastructure issue is much harder to debug than an
    issue discovered by your own monitoring.

Google's existing monitoring tools didn't work well in Cloud, necessitating the
need to build things from ground up. And since probers are the cornerstone of
monitoring and reliability at Google[^2], that's where we decided to start.
_Thus began the journey of Cloudprober_.

[^2]:
    Almost all of Google's systems rely on probers to detect customer facing
    problems.

Even though the primary goal of Cloudprober at that time was to discover and
alert on Cloud Networking availability and performance problems, we decided to
develop it as a generic prober that could be used to monitor a wide variety of
systems and services. We also decided to make Cloudprober open source so that a
wider community could trust it, contribute to it, and run it on their own
systems.

## Scale, Efficiency

For scales as big as Google Cloud, horizontal scalability and efficiency become
critical requirements, and for a monitoring software to be useful reliability is
super important as well. Keeping these requirements in mind, our goal for
Cloudprober was for it to be able to reliably monitor 100s of 1000s of endpoints
(IPs, Ports, HTTP/S URLs, etc) from each instance, while keeping the resource
requirements and management overhead very low[^3].

[^3]:
    Hostinger was able to probe 1.8M targets using a single instance:
    [blog](https://www.hostinger.com/blog/cloudprober-explained-the-way-we-use-it-at-hostinger).

Cloudprober maximizes resources utilization by relying heavily on Go concurrency
(_resource efficiency_), supports probing large number of targets in parallel at
a high frequency (_each instance does more_), minimizes the need of frequent
updates by supporting dynamic targets discovery (_ease of management_), has
native implementations for common probe types (_efficiency_), and so on.

## Beyond Google and Open-Source

We
[open-sourced](https://opensource.googleblog.com/2018/03/cloudprober-open-source-black-box.html)
Cloudprober in 2017. That brought in a new phase in its evolution. We added many
features over time to make it more useful to the wider community, such as
first-class Kubernetes support, a built-in probe status UI, PostgreSQL and
Cloudwatch surfacers, OAuth support, Validators, and most recently, built-in
alerting capability.

We used the same codebase for the internal and open-source versions, which was
more work but it created a huge advantage -- our own extensive internal
deployment provided a continuous testing platform for Cloudprober, particularly
for its scalability and performance aspects, while we added all these features.

## Move away from Google Github

I left Google in Nov 2021. To keep working on Cloudprober independently, I moved
Cloudprober's Github repository from
<a href="https://github.com/google/cloudprober">github.com/google/cloudprober<a>
to
<a href="https://github.com/cloudprober/cloudprober">github.com/cloudprober/cloudprober</a>.
This was a disruptive move and we lost a lot of Github stars in the process
(1.4k - :smiley:), but overall it was a good move as Cloudprober has grown much
faster after becoming independent.

While I can't say this authoritatively now as I don't work there anymore, from
what I know, Google still uses Cloudprober, in fact, even more widely now.

## Growth and stability

Throughout its journey, Cloudprober has continuously adapted and expanded to
meet the evolving needs of its users[^4]. To ensure that Cloudprober thrives and
evolves robustly, we've been very diligent that it grows in a structured way, a
commitment we'll uphold in future as well.

[^4]:
    I think it's an essential trait for any software. Software that don't evolve
    with time wither away.
