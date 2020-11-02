# Creating your very first Go Operator using the Operator Framework SDK

## Installing the Operator SDK

~~~sh
RELEASE_VERSION=v1.1.0
# Linux
sudo curl -L https://github.com/operator-framework/operator-sdk/releases/download/${RELEASE_VERSION}/operator-sdk-${RELEASE_VERSION}-x86_64-linux-gnu -o /usr/local/bin/operator-sdk
# macOS
sudo curl -L https://github.com/operator-framework/operator-sdk/releases/download/${RELEASE_VERSION}/operator-sdk-${RELEASE_VERSION}-x86_64-apple-darwin -o /usr/local/bin/operator-sdk
# Linux / macOS
sudo chmod +x /usr/local/bin/operator-sdk
~~~

## Initialize the Operator Project

~~~sh
mkdir -p ~/operators-projects/reverse-words-operator && cd $_
export GO111MODULE=on
export GOPROXY=https://proxy.golang.org
export GH_USER=<your_github_user>
go get github.com/operator-framework/operator-lib@v0.1.0
operator-sdk init --domain=linuxera.org --repo=github.com/$GH_USER/reverse-words-operator
~~~

## Create the Operator API Types

~~~sh
operator-sdk create api --group=apps --version=v1alpha1 --kind=ReverseWordsApp --resource=true --controller=true
curl -Ls https://raw.githubusercontent.com/emerging-tech-vlc/operators-everywhere/master/go-operator/files/reversewordsapp_types.go -o ~/operators-projects/reverse-words-operator/api/v1alpha1/reversewordsapp_types.go
# Generate  boilerplate
make generate
~~~

## Add a Controller to the Operator

~~~sh
curl -Ls https://raw.githubusercontent.com/emerging-tech-vlc/operators-everywhere/master/go-operator/files/reversewordsapp_controller.go -o ~/operators-projects/reverse-words-operator/controllers/reversewordsapp_controller.go
sed -i "s/GHUSER/${GH_USER}/" ~/operators-projects/reverse-words-operator/controllers/reversewordsapp_controller.go
~~~

## Setup Watch Namespaces

~~~sh
curl -Ls https://raw.githubusercontent.com/emerging-tech-vlc/operators-everywhere/master/go-operator/files/main.go -o ~/operators-projects/reverse-words-operator/main.go
sed -i "s/GHUSER/${GH_USER}/" ~/operators-projects/reverse-words-operator/main.go
~~~

## Create the required manifest for deploying the operator

~~~sh
make manifests
~~~

## Run the tests

~~~sh
# Setup EnvTest
curl https://raw.githubusercontent.com/kubernetes-sigs/controller-runtime/v0.6.2/hack/setup-envtest.sh -o /tmp/setup-envtest.sh
source /tmp/setup-envtest.sh
fetch_envtest_tools ~/operators-projects/reverse-words-operator/testbin
set +o errexit
set +o pipefail

# Run the tests
export KUBEBUILDER_ASSETS=~/operators-projects/reverse-words-operator/testbin/bin
make test
~~~

## Build the Operator

~~~sh
export QUAY_USER=<your_quay_user>
export KUBEBUILDER_ASSETS=~/operators-projects/reverse-words-operator/testbin/bin
# If using podman edit Makefile to use podman instead of docker
make docker-build docker-push IMG=quay.io/$QUAY_USER/reversewords-operator:v0.0.1
~~~

## Generate the Operator Bundle

An Operator Bundle consists of different manifests (CSVs and CRDs) and some metadata that defines the Operator at a specific version.

~~~sh
make bundle VERSION=0.0.1 CHANNELS=alpha DEFAULT_CHANNEL=alpha IMG=quay.io/$QUAY_USER/reversewords-operator:v0.0.1
curl -Ls https://raw.githubusercontent.com/emerging-tech-vlc/operators-everywhere/master/go-operator/files/reverse-words-operator.clusterserviceversion_v0.0.1.yaml -o ~/operators-projects/reverse-words-operator/bundle/manifests/reverse-words-operator.clusterserviceversion.yaml
sed -i "s/QUAY_USER/$QUAY_USER/g" ~/operators-projects/reverse-words-operator/bundle/manifests/reverse-words-operator.clusterserviceversion.yaml
~~~

## Review the CSV

Go to https://operatorhub.io/preview and paste the CSV content from file `reverse-words-operator.clusterserviceversion.yaml`

## Deploy the CSV

### Publish our own Bundle

1. Build the bundle
2. Push and validate the bundle
3. Create the Index Image

Index Image is an image which contains a database of pointers to operator manifest content that is easily queriable via an included API that is served when the container image is run.

~~~sh
# Build the bundle
podman build -f bundle.Dockerfile -t quay.io/$QUAY_USER/reversewords-operator-bundle:v0.0.1
# Push and validate the bundle
podman push quay.io/$QUAY_USER/reversewords-operator-bundle:v0.0.1
operator-sdk bundle validate quay.io/$QUAY_USER/reversewords-operator-bundle:v0.0.1 -b podman
# Create the Index image
sudo curl -sL https://github.com/operator-framework/operator-registry/releases/download/v1.15.0/linux-amd64-opm -o /usr/local/bin/opm && chmod +x /usr/local/bin/opm
opm index add -c podman --bundles quay.io/$QUAY_USER/reversewords-operator-bundle:v0.0.1 --tag quay.io/$QUAY_USER/reversewords-index:v0.0.1
podman push quay.io/$QUAY_USER/reversewords-index:v0.0.1
~~~

### Load the CatalogSource onto the cluster

At this point we have our bundle and index image ready, we just need to create the required CatalogSource into the cluster so we get access to our Operator bundle.

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
  image: quay.io/$QUAY_USERNAME/reversewords-index:v0.0.1
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
   make bundle VERSION=0.0.2 CHANNELS=alpha DEFAULT_CHANNEL=alpha IMG=quay.io/$QUAY_USERNAME/reversewords-operator:v0.0.2
   ~~~
3. Tweak the CSV with proper installModes etc.
   
   ~~~sh
   curl -Ls https://raw.githubusercontent.com/emerging-tech-vlc/operators-everywhere/master/go-operator/reverse-words-operator.clusterserviceversion_v0.0.2.yaml -o ~/operators-projects/reverse-words-operator/bundle/manifests/reverse-words-operator.clusterserviceversion.yaml
   sed -i "s/QUAY_USER/$QUAY_USERNAME/g" ~/operators-projects/reverse-words-operator/bundle/manifests/reverse-words-operator.clusterserviceversion.yaml
   ~~~
4. Build the new bundle

   ~~~sh
   podman build -f bundle.Dockerfile -t quay.io/$QUAY_USERNAME/reversewords-operator-bundle:v0.0.2
   ~~~
5. Push and validate the new bundle

   ~~~sh
   podman push quay.io/$QUAY_USERNAME/reversewords-operator-bundle:v0.0.2
   operator-sdk bundle validate quay.io/$QUAY_USERNAME/reversewords-operator-bundle:v0.0.2 -b podman
   ~~~
6. Update the Index image

   ~~~sh
   # Create the index image
   opm index add -c podman --bundles quay.io/$QUAY_USERNAME/reversewords-operator-bundle:v0.0.2 --from-index quay.io/$QUAY_USERNAME/reversewords-index:v0.0.1 --tag quay.io/$QUAY_USERNAME/reversewords-index:v0.0.2
   # Push the index image
   podman push quay.io/$QUAY_USERNAME/reversewords-index:v0.0.2
   ~~~
7. Patch the catalogsource

   ~~~sh
   PATCH="{\"spec\":{\"image\":\"quay.io/$QUAY_USERNAME/reversewords-index:v0.0.2\"}}"
   oc -n openshift-marketplace patch catalogsource emergingtech-catalog -p $PATCH --type merge
   ~~~
8. Now you can update your operators modifying the subscription or if the auto update is enabled, they will be updated automatically
