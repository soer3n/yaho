+++
title = "Releases"
chapter = false
weight = 30
+++

#### Release

> The release resource represents a helm release and is comparable to helm cli command "helm upgrade --install ...". It maps a release installation and/or upgrade process. See [here](https://github.com/soer3n/apps-operator/blob/master/apis/yaho/v1alpha1/release_types.go) for detailed information about the spec structure. You cannot define values directly in the release resource. This is solved by an own values resource which is explained [here](/architecture/value).

{{< mermaid >}}
%%{init:{"theme":"forest", "mirrorActors":"true", "sequence": {"showSequenceNumbers":false, "wrap": false,"useMaxWidth": true}}}%%
sequenceDiagram
    participant C AS client
    participant R AS reconciler
    participant RM AS release model
    participant CVM AS chart version model
    participant VM AS value model
    participant PKG AS helm v3 go.pkg
    participant K AS kube-apiserver
    C->>K: create/update chart resource
    K-->>C: return error
    R->>RM: init model
    RM->>PKG: init action configuration
    rect rgb(255, 255, 204)
    alt spec.config != nil
        RM->>K: load config resource
        RM-->>RM: set options
    end
    end
    RM->>VM: init value model
    VM->>K: get list of related value resources
    VM-->>VM: transform values
    VM-->>RM: return value model
    RM->>K: get default value configmap
    rect rgb(255, 255, 204)
    alt err != nil
        K-->>R: return error
    end
    end
    RM->>K: get chart index configmap
    rect rgb(255, 255, 204)
    alt err != nil
        K-->>R: return error
    end
    end
    RM->>K: get chart custom resource
    rect rgb(255, 255, 204)
    alt err != nil
        K-->>R: return error
    end
    end
    RM->>CVM: init chart version model
    CVM-->>CVM: parse version
    CVM->>K: get respository resource
    rect rgb(255, 255, 204)
    alt err != nil
        K-->>R: return error
    end
    end
    CVM->>PKG: resolve reference url
    rect rgb(255, 255, 204)
    alt values == nil
        VM->>K: get default value configmap
        rect rgb(255, 255, 204)
        alt err != nil
            K-->>R: return error
        end
        end
    end
    end
    CVM-->>CVM: load dependencies
    Note right of CVM: load dependencies means to iterate over index deps<br><br> and init a chartversion model for each item
    rect rgb(191, 223, 255)
    loop iterate over chart and dependencies
        RM->>PKG: validate chart
        rect rgb(255, 255, 204)
        alt error != nil
            PKG-->>R: return error
        end
        end
    end
    end
    R-->>R: handle finalizer
    rect rgb(255, 255, 204)
    alt requeue
        rect rgb(255, 255, 204)
        alt markedToBeDeleted
            R->>RM: remove release
            RM->>PKG: init remove client
            RM->>PKG: remove release
            RM-->>R: return for reconciling
        end
        end
        R->>K: update release resource
        RM-->>R: return for reconciling
    end
    end
    R->>RM: update release
    RM->>PKG: init get client
    RM->>PKG: get existing release
    rect rgb(255, 255, 204)
    alt err != nil
        PKG-->>R: return error
    end
    end
    rect rgb(255, 255, 204)
    alt release != nil
        RM->>PKG: init get values client
        RM->>PKG: get installed values
        rect rgb(255, 255, 204)
        alt err != nil
            PKG-->>R: return error
        end
        end
        RM-->>RM: compare values
        rect rgb(255, 255, 204)
        alt valuesChanged
            RM->>PKG: init upgrade client
            RM-->>RM: set upgrade flags
            RM->>PKG: upgrade release
            rect rgb(255, 255, 204)
            alt err != nil
                PKG-->>R: return error
            end
            end
        end
        end
        RM-->>R: return
    end
    end
    RM->>PKG: init install client
    RM-->>RM: set install flags
    RM->>PKG: install release
    rect rgb(255, 255, 204)
    alt err != nil
        PKG-->>R: return error
    end
    end
{{< /mermaid >}}

\
\
#### ReleaseGroups

> The releasegroup resource represents a collection of helm releases. The idea behind is to control releases which have dependencies to each other. At the moment it's just a collection without logic for managing them together. In general it deploys a collection of release resources. See [here](https://github.com/soer3n/apps-operator/blob/master/apis/yaho/v1alpha1/releasegroup_types.go) for detailed information about the spec structure.

{{< mermaid >}}
%%{init:{"theme":"forest", "mirrorActors":"true", "useMaxWidth":"true", "sequence": {"showSequenceNumbers":false, "wrap": true, "width":350}, "sequenceConfig": {
    "diagramMarginX": 50,
    "diagramMarginY": 10,
    "boxTextMargin": 5,
    "noteMargin": 10,
    "messageMargin": 35,
    "mirrorActors": true
}}}%%
sequenceDiagram
    participant C AS client
    participant R AS reconciler
    participant K AS kube-apiserver
    C->>K: create/update release group object
    R->>K: get releases by labels
    rect rgb(191, 223, 255)
    loop returned release resources
        rect rgb(255, 255, 204)
        alt item is not in spec list
        R->>K: remove release resource
        end
        end
    end
    end
    rect rgb(191, 223, 255)
    loop specified release resources
        R->>K: create/update release resource
    end
    end
{{< /mermaid >}}
