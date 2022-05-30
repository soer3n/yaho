+++
title = "Configuration"
weight = 20
chapter = true
+++

> There are some environment variables for general settings which can be set at runtime.

| Name               | Default | Description                                    |
|--------------------|---------|------------------------------------------------|
| WATCH_NAMESPACE    | default | Namespace for creating and watching configmaps |
| SCAN_INTERVAL      | 10s     | Interval for scanning remote repository        |
| REPO_SYNC_ENABLED  | true    | Enables sync for repository related configmaps |
| CHART_SYNC_ENABLED | true    | Enables sync for chart related configmaps      |
| AUTO_CREATE_CHARTS | true    | Enables auto installation for charts           |
