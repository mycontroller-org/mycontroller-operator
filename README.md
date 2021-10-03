# mycontroller-operator

MyController operator to manage [MyController](https://v2.mycontroller.org/) server in Kubernetes/OpenShift environment.

## Deploy in Kubernetes
```bash
make deploy
```

## Deploy in OpenShift
#### Catalog Source (mycontroller-catalog-source.yaml)
```yaml
apiVersion: operators.coreos.com/v1alpha1
kind: CatalogSource
metadata:
  name: mycontroller-cs
  namespace: openshift-marketplace
spec:
  displayName: MyController Catalog
  image: quay.io/mycontroller/mycontroller-operator-catalog:v0.0.1
  sourceType: grpc
```

#### Subscription (mycontroller-subscription.yaml)
```yaml
apiVersion: operators.coreos.com/v1alpha1
kind: Subscription
metadata:
  name: mycontroller-operator
  namespace: openshift-operators
spec:
  channel: alpha
  installPlanApproval: Automatic
  name: mycontroller-operator
  source: mycontroller-cs
  sourceNamespace: openshift-marketplace
```
```bash
oc create -f mycontroller-catlog-source.yaml
oc create -f mycontroller-subscription.yaml
```

## Deploy MyController server
```bash
# basic install
kubectl create -f https://raw.githubusercontent.com/mycontroller-org/mycontroller-operator/master/config/samples/v1-basic-install.yaml

# with storage
kubectl create -f https://raw.githubusercontent.com/mycontroller-org/mycontroller-operator/master/config/samples/v1-with-storage.yaml
```