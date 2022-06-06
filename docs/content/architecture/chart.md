+++
title = "Charts"
chapter = false
weight = 20
+++

#### Chart

> The chart resource represents the specification for an helm chart. For every specified and valid version the controller will create or update  3 configmaps for chart templates, crds and default values. If enabled dependency chart resources will be also created or updated. See [here](https://github.com/soer3n/apps-operator/blob/master/apis/yaho/v1alpha1/chart_types.go) for detailed information about the spec structure.

{{< mermaid >}}
%%{init:{"theme":"forest", "mirrorActors":"true", "sequence": {"showSequenceNumbers":false, "wrap": false,"useMaxWidth": true}}}%%
sequenceDiagram
    participant C AS client
    participant R AS reconciler
    participant M AS chart model
    participant VM AS chartversion model
    participant PKG AS helm v3 go.pkg
    participant K AS kube-apiserver
    participant H AS helm repository
    C->>K: create/update chart resource
    R->>M: init model
    M-->>M: set metadata
    M->>K: load index configmap
    rect rgb(191, 223, 255)
    loop iterate over specified versions
        M->>VM: init chartversion model
        VM-->>VM: parse version
        VM->>K: get respository resource
        rect rgb(255, 255, 204)
        alt error != nil
            K-->>M: return error
        end
        end
        VM->>PKG: resolve reference url
        rect rgb(255, 255, 204)
        alt values == nil
            VM->>K: get default value configmap
            rect rgb(255, 255, 204)
            alt error != nil
                K-->>M: return error
            end
            end
        end
        end
        VM-->>VM: load dependencies
        Note right of VM: load dependencies means to iterate over index deps<br><br> and init a chartversion model for each item
        VM-->>M: return object
        M-->>M: append version object
    end
    end
    M-->>R: return object
    R-->>M: create/update chart resource
    rect rgb(191, 223, 255)
    loop iterate over specified versions
        M->>VM: prepare version object
        rect rgb(255, 255, 204)
        alt object == nil
            VM->>H: download chartversion
            rect rgb(255, 255, 204)
            alt error != nil
                H-->>M: return error
            end
            end
            H-->>VM: return object
            VM->>PKG: loadArchive
            rect rgb(255, 255, 204)
            alt error != nil
                H-->>M: return error
            end
            end
            PKG-->>VM: return unmarshaled chart object
        end
        end
        M->>VM: manage configmaps related to chartversion
        rect rgb(191, 223, 255)
        loop iterate over specified version
            VM->>K: create/update configmaps
            rect rgb(255, 255, 204)
            alt error != nil
                K-->>M: return error
            end
            end
        end
        end
    end
    end
    R-->>M: create/update dependency charts
    rect rgb(191, 223, 255)
    loop iterate over object versions
        M-->>VM: manage subresources
        rect rgb(191, 223, 255)
        loop iterate over object version dependencies
            VM->>K: create/update chart resource
             rect rgb(255, 255, 204)
            alt error != nil
                K-->>M: return error
            end
            end
        end
        end
    end
    end
{{< /mermaid >}}
