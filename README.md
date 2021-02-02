# PDB Pods Owners Discovery

Simple CLI that displays the owning top-level Kubernetes resources (e.g. Deployments, StatefulSets, ...) of PodDisruptionBudgets.

## Usage

The CLI will use your usual `.kubeconfig` file to authenticate against a Kubernetes cluster.

```shell
$ ./pdb-pods-owners-discovery
+-----------------------------------------------------------------------------+
| Kubernetes resources impacted by active PodDisruptionBudgets                |
+-------------------------------+-----------+---------------------+-----------+
| NAME                          | NAMESPACE | GROUP VERSION KIND  | PDB       |
+-------------------------------+-----------+---------------------+-----------+
| argocd-dex-server             | argocd    | apps/v1/Deployment  | test-aac2 |
| argocd-redis                  | argocd    | apps/v1/Deployment  | test-aac2 |
| argocd-repo-server            | argocd    | apps/v1/Deployment  | test-aac2 |
| argocd-server                 | argocd    | apps/v1/Deployment  | test-aac2 |
| private-git-repository        | argocd    | apps/v1/Deployment  | test-aac2 |
| argocd-application-controller | argocd    | apps/v1/StatefulSet | test-aac2 |
+-------------------------------+-----------+---------------------+-----------+
```
