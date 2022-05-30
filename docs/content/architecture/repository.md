+++
title = "Repositories"
chapter = true
weight = 10
+++

> The repo resource represents an initialization of an helm repository. It is similar to helm cli command "helm repo add ..." and downloads the file for parsing charts which are part of requested repository. It is also parsing the chart resources. See [here](https://github.com/soer3n/apps-operator/blob/master/apis/helm/v1alpha1/repo_types.go) for detailed information about the spec structure.

{{< mermaid align="left" >}}
%%{init:{"theme":"forest", "mirrorActors":"true", "sequence": {"showSequenceNumbers":false, "wrap": false,"useMaxWidth": true}}}%%
sequenceDiagram
    participant C AS client
    participant R AS reconciler
    participant M AS model
    participant K AS kube-apiserver
    participant H AS helm repository
    C->>K: create/update repository object
    R->>M: init model
    M->>H: download repository
    H-->>M: return response
    M-->>R: return object
    rect rgb(191, 223, 255)
    loop create/update chart indices
        M->>K: create/update chart index cm
    end
    end
    R->>M: create/update specified chart objects
    rect rgb(191, 223, 255)
    loop create/update chart resources
        M->>K: create/update chart custom resource
        rect rgb(255, 255, 204)
        alt error != nil
            K-->>R: return error
        end
        end
    end
    end
    # Note right of H: Rational thoughts <br/>prevail...
    
{{< /mermaid  >}}


### RepoGroups

> The repogroup resource represents a collection of helm repositories. This is needed if you want to deploy an helm release which has dependency charts which are part of different repositories. If dependencies are part of the same repository you don't need this. See [here](https://github.com/soer3n/apps-operator/blob/master/apis/helm/v1alpha1/repogroup_types.go) for detailed information about the spec structure.

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
    C->>K: create/update repository group object
    R->>K: get repositories by labels
    rect rgb(191, 223, 255)
    loop returned repository resources
        rect rgb(255, 255, 204)
        alt item is not in spec list
        R->>K: remove repository resource
        end
        end
    end
    end
    rect rgb(191, 223, 255)
    loop specified repository resources
        R->>K: create/update repository resource
    end
    end
{{< /mermaid >}}
