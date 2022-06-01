+++
title = "Values"
chapter = false
weight = 40
+++

#### Values

>  values resource represents in general a values file for a release. There is some own logic there. The resource is splitted into two parts. The values and references to another values spec. See [here](https://github.com/soer3n/apps-operator/blob/master/apis/helm/v1alpha1/values_types.go) for detailed information about the spec structure. The idea here is that these resources are managed like a construction kit for handling values for different releases. The main benefits are that you can stretch your values structure for a single release and that you can connect similar configurations for different releases. An example would be the definition of resource requests and limits.

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
    C->>K: create/update value object
    rect rgb(191, 223, 255)
    loop release name in annotations
        rect rgb(255, 255, 204)
        alt release.Status.Synced
            R->>K: update release resource
            rect rgb(255, 255, 204)
            alt err != nil
                R-->>R: return for reconciling
            end
            end
        end
        end
    end
    end
{{< /mermaid >}}
