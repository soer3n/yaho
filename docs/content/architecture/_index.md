+++
title = "Architecture"
weight = 15
chapter = true
+++

#### Overview

In general the procedure is nearly similar to well-known helm cli. At first you've to deploy a repository or a group of repositories. Charts which index should be loaded can be specified in a repository resource and also by creating a chart resource manually. If every chart which is needed for a helm release, which includes dependency charts, is present a release resource can be created and the operator will install the defined release with specified chart version and values which custom resources can be chained.

![Alt text](/general.drawio.png?raw=true "Overview")


#### Resources

For detailed information about the workflow of the custom resources go to the related subpage.
