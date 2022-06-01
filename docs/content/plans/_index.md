+++
title = "Plans & ToDos"
weight = 35
chapter = true
+++

## Plans

- add multicluster functionalitiy for releases (API change)
- add git as a source for repository and chart resources (API change)
- add custom resource for helm plugin configuration (API change)
- add job to migrate from helm managed release to operator custom resources 

&nbsp;

## ToDos

- do not install index configmaps when charts not set in repository resource
- set status to failed if repository couldn't be found for a dependency chart
- add mutating admission webhook for resource manipulation
- split into source & release controller
- improve group concepts for repositories and releases
- handle embedded goroutines with contexts
- syncing state of releases continiously (check if there changes due to manual actions)
- switching to previous revision and back
- auto-sync for new chart versions from repository
- black- and whitelisting for charts when auto-sync for repository is enabled
