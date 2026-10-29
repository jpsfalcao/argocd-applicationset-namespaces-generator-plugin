# argocd-applicationset-namespaces-generator-plugin

Namespaces Generator that discovers namespaces in a target cluster.

It can be used as ArgoCD ApplicationSet plugin https://argo-cd.readthedocs.io/en/stable/operator-manual/applicationset/Generators-Plugin/.

It can discover existing namespaces in the cluster to produce an app per each namespace.

# Output

Each namespace produces its name and labels. `labels` is always an object,
never `null`. Annotations are not returned.

```json
{
  "namespace": "team-a",
  "labels": { "kubernetes.io/metadata.name": "team-a", "example.com/tier": "s" }
}
```

With `goTemplate: true`, read labels with `index`, since label keys contain
dots and slashes: `{{ index .labels "example.com/tier" }}`. With
`goTemplate: false`, Argo CD flattens them to `labels.<key>` strings.

`index` on a missing key renders `<no value>` without an error, even with
`goTemplateOptions: ["missingkey=error"]`. Make sure every namespace the
generator returns has the labels you use, either with the `labelSelector`
(`Exists` or `In`) or with an explicit check in the template:

```yaml
'{{ if not (hasKey .labels "example.com/tier") }}{{ fail "missing tier label" }}{{ end }}'
```

Label values come from whoever can label namespaces, so treat them as input:
restrict accepted values with the `labelSelector` (`In`), and control who may
set the labels (RBAC or an admission policy).

# Assumptions and prerequisites

- You are using JWT authentication to your clusters (i.e. Downward API tokens mounted to pods)
- If using external clusters, you must populate cluster annotation with its Certificate Authority
- A bearer token is required (`--listen-token` or `ARGOCD_APPLICATIONSET_NAMESPACES_PLUGIN_LISTEN_TOKEN`); the server refuses to start without one, except with `--local`

# Usage

## Local

```bash
go run ./... -v=4 --log-format=json server --local
curl -X POST -H "Content-Type: application/json" -d @testdata/request.json http://localhost:8080/api/v1/getparams.execute
```

## In Cluster

1. Access the ArgoCD GUI.

   ```console
   argocd admin dashboard -n argocd
   ```

2. Open the browser http://localhost:8080/

3. Deploy the argocd-applicationset-namespaces-generator-plugin Deployment.

   ```console
   kubectl apply -f testdata/manifest.yaml
   ```

4. Deploy the ApplicationSet YAMLs.

   ```console
   kubectl apply -f testdata/appset-project-hsiaoairplane-namespaces-rbac.yaml

   kubectl apply -f testdata/appset-project-airplanehsiao-namespaces-rbac.yaml

   kubectl apply -f testdata/appset-projects-namespaces-rbac.yaml
   ```

5. Create ArgoCD ApplicationProjects.

   ```console
   argocd proj create project-hsiaoairplane -s "*" --dest "*,*" --allow-namespaced-resource "rbac.authorization.k8s.io/Role" --allow-namespaced-resource "rbac.authorization.k8s.io/RoleBinding" --allow-namespaced-resource "*/ServiceAccount" --upsert

   argocd proj create project-airplanehsiao -s "*" --dest "*,*" --allow-namespaced-resource "rbac.authorization.k8s.io/Role" --allow-namespaced-resource "rbac.authorization.k8s.io/RoleBinding" --allow-namespaced-resource "*/ServiceAccount" --upsert

   argocd proj create projects -s "*" --dest "*,*" --allow-namespaced-resource "rbac.authorization.k8s.io/Role" --allow-namespaced-resource "rbac.authorization.k8s.io/RoleBinding" --allow-namespaced-resource "*/ServiceAccount" --upsert
   ```

6. Create the project "hsiaoairplane" namespaces.

   ```console
   kubectl create namespace foo
   kubectl label namespace foo project=hsiaoairplane --overwrite=true

   kubectl create namespace foobar
   kubectl label namespace foobar project=hsiaoairplane --overwrite=true

   kubectl create namespace hsiaoairplane
   kubectl label namespace hsiaoairplane project=hsiaoairplane --overwrite=true
   ```

7. Create the project "airplanehsiao" namespaces.

   ```console
   kubectl create namespace bar
   kubectl label namespace bar project=airplanehsiao --overwrite=true

   kubectl create namespace barfoo
   kubectl label namespace barfoo project=airplanehsiao --overwrite=true

   kubectl create namespace airplanehsiao
   kubectl label namespace airplanehsiao project=airplanehsiao --overwrite=true
   ``` 
