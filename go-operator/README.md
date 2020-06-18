# Creating your very first Go Operator using the Operator Framework SDK

## Installing the Operator SDK

~~~sh
RELEASE_VERSION=v0.18.1
# Linux
sudo curl -L https://github.com/operator-framework/operator-sdk/releases/download/${RELEASE_VERSION}/operator-sdk-${RELEASE_VERSION}-x86_64-linux-gnu -o /usr/local/bin/operator-sdk
# macOS
sudo curl -L https://github.com/operator-framework/operator-sdk/releases/download/${RELEASE_VERSION}/operator-sdk-${RELEASE_VERSION}-x86_64-apple-darwin -o /usr/local/bin/operator-sdk
# Linux / macOS
sudo chmod +x /usr/local/bin/operator-sdk
~~~

## Initialize the Operator Project

~~~sh
mkdir -p ~/operators-projects/ && cd $_
export GO111MODULE=on
export GOPROXY=https://proxy.golang.org
export GH_USER=<your_github_user>
operator-sdk new reverse-words-operator --repo github.com/${GH_USER}/reverse-words-operator
cd reverse-words-operator
~~~

## Create the Operator API Types

~~~sh
operator-sdk add api --api-version=emergingtech.vlc/v1alpha1 --kind=ReverseWordsApp
curl -Ls https://raw.githubusercontent.com/emerging-tech-vlc/operators-everywhere/master/go-operator/files/reversewordsapp_types.go -o pkg/apis/emergingtech/v1alpha1/reversewordsapp_types.go
operator-sdk generate k8s
~~~

## Generate the Validation section in the CRD

~~~sh
operator-sdk generate crds
~~~

## Add a Controller to the Operator

~~~sh
operator-sdk add controller --api-version=emergingtech.vlc/v1alpha1 --kind=ReverseWordsApp
curl -Ls https://raw.githubusercontent.com/emerging-tech-vlc/operators-everywhere/master/go-operator/files/reversewordsapp_controller.go -o pkg/controller/reversewordsapp/reversewordsapp_controller.go
sed -i "s/GHUSER/${GH_USER}/" pkg/controller/reversewordsapp/reversewordsapp_controller.go
~~~

## Build the Operator

~~~sh
export QUAY_USER=<your_quay_user>
operator-sdk build quay.io/${QUAY_USER}/reverse-words-operator:v0.1.0 --image-builder podman
podman push quay.io/${QUAY_USER}/reverse-words-operator:v0.1.0
~~~

## Generate the Cluster Service Version 

~~~sh
operator-sdk olm-catalog gen-csv --csv-version 0.1.0 --update-crds
curl -Ls https://raw.githubusercontent.com/emerging-tech-vlc/operators-everywhere/master/go-operator/files/reverse-words-operator.v0.1.0.clusterserviceversion.yaml -o deploy/olm-catalog/reverse-words-operator/0.1.0/reverse-words-operator.v0.1.0.clusterserviceversion.yaml
sed -i "s/QUAYUSER/${QUAY_USER}/" deploy/olm-catalog/reverse-words-operator/0.1.0/reverse-words-operator.v0.1.0.clusterserviceversion.yaml
~~~

## Review the CSV

Go to https://operatorhub.io/preview and paste the CSV content from file `reverse-words-operator.v0.1.0.clusterserviceversion.yaml`

## Deploy the CSV

### Create our own Catalog of Operators

We are going to build our own `Catalog of Operators` so once published on the cluster we don't need to load CSVs manually.

1. Clone the [operator-registry](https://github.com/operator-framework/operator-registry) repository
2. Get the package file from the operator
3. Copy CSV + Package + Versioned CRDs (Operator Bundle) to the `manifests` folder
4. Build the Registry

~~~sh
# Clone the operator-registry
cd ~/operators-projects/
git clone https://github.com/operator-framework/operator-registry
cd operator-registry
# Clean pre-existing operator bundles
rm -rf manifests/*
# Copy the operator bundle
cp -r ~/operators-projects/reverse-words-operator/deploy/olm-catalog/reverse-words-operator manifests/
# Build the Registry
podman build -f upstream-example.Dockerfile -t quay.io/${QUAY_USER}/emergingtech-catalog:v1
# Push the registry
podman push quay.io/${QUAY_USER}/emergingtech-catalog:v1
~~~

### Load the CatalogSource onto the cluster

~~~sh
cat <<EOF | oc -n openshift-marketplace create -f -
kind: CatalogSource
apiVersion: operators.coreos.com/v1alpha1
metadata:
  name: emergingtech-catalog
spec:
  sourceType: grpc
  displayName: Emerging Tech Operators
  publisher: Emerging Tech Valencia
  image: quay.io/${QUAY_USER}/emergingtech-catalog:v1
EOF
~~~

After a few seconds we should see the `PackageManifest` for our Operator loaded:

~~~sh
oc -n openshift-marketplace get packagemanifest reverse-words-operator
NAME                     CATALOG                   AGE
reverse-words-operator   Emerging Tech Operators   3m22s
~~~

### Deploy our Operator

Now we can create a subscription using the CLI or using the WebUI.

**CLI**

~~~sh
NAMESPACE=my-namespace
cat <<EOF | oc -n $NAMESPACE create -f - 
apiVersion: operators.coreos.com/v1alpha1
kind: Subscription
metadata:
  name: reversewords-subscription
spec:
  channel: alpha
  name: reverse-words-operator
  installPlanApproval: Automatic
  source: emergingtech-catalog
  sourceNamespace: openshift-marketplace
---
apiVersion: operators.coreos.com/v1
kind: OperatorGroup
metadata:
  name: reverse-words-operatorgroup
spec:
  targetNamespaces:
  - $NAMESPACE
EOF
~~~

**WebUI**

Go to the OperatorHub catalog and search the operator within the WebUI.

### Publish and update for our Operator

1. Generate a new CSV version

   ~~~sh
   cd ~/operators-projects/reverse-words-operator
   operator-sdk olm-catalog gen-csv --csv-version 0.2.0 --from-version 0.1.0 --update-crds
   ~~~
2. Update the Catalog of Operators
   
   ~~~sh
   cd ~/operators-projects/operator-registry
   # Copy the operator bundle
   cp -r ~/operators-projects/reverse-words-operator/deploy/olm-catalog/reverse-words-operator manifests/
   ~~~
3. Update the Package file

   ~~~sh
   cat <<EOF > manifests/reverse-words-operator/reverse-words-operator.package.yaml
   channels:
   - name: alpha
     currentCSV: reverse-words-operator.v0.1.0
   - name: beta
     currentCSV: reverse-words-operator.v0.2.0
   - name: stable
     currentCSV: reverse-words-operator.v0.1.0
   defaultChannel: alpha
   packageName: reverse-words-operator
   EOF
   ~~~
4. Build and push a new version of the Catalog

   ~~~sh
   # Build the Registry
   podman build -f upstream-example.Dockerfile -t quay.io/${QUAY_USER}/emergingtech-catalog:v2
   # Push the registry
   podman push quay.io/${QUAY_USER}/emergingtech-catalog:v2
   ~~~
5. Update the CatalogSource

   ~~~sh
   PATCH={"spec":{"image":"quay.io/${QUAY_USER}/emergingtech-catalog:v2"}}
   oc -n openshift-marketplace patch catalogsource emergingtech-catalog -p '$PATCH' --type merge
   ~~~
6. Now you can update your operators modifying the subscription
