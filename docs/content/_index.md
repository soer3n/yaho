+++
title = "Yet Another Helm Operator"
weight = 5
chapter = true
+++

## Introduction

This project tries to picture helm as an kubernetes api extension. This operator manages helm repositories, charts, releases and values in a declarative way.

#### reusing values

Through a custom resource for values reusing of them in different releases with same sub specifications is one feature. Go to [values](architecture/value) for details.

#### permission management

Another feature is to use kubernetes rbac and a resource for configuration to restrict and configure the usage in a kubernetes cluster. Go to [configuration](configuration) for details.

#### shared sources

Files, metadata and default values are stored as binary data in configmaps. So there are no local files which could differ in multiple workstations. Go to [architecture](architecture) for details.
