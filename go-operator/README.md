# Creating your very first Go Operator using the Operator Framework SDK



## Installing the Operator SDK

~~~sh
RELEASE_VERSION=v0.13.0
# Linux
curl -L https://github.com/operator-framework/operator-sdk/releases/download/${RELEASE_VERSION}/operator-sdk-${RELEASE_VERSION}-x86_64-linux-gnu -o /usr/local/bin/operator-sdk
# macOS
curl -L https://github.com/operator-framework/operator-sdk/releases/download/${RELEASE_VERSION}/operator-sdk-${RELEASE_VERSION}-x86_64-apple-darwin -o /usr/local/bin/operator-sdk
# Linux / macOS
chmod +x /usr/local/bin/operator-sdk
~~~

## Initialize the Operator Project

~~~sh
mkdir -p ~/operators-projects/ && cd $_
operator-sdk new reverse-words-operator --repo=github.com/<github_user>/reverse-words-operator
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
~~~

## Build the Operator

~~~sh
operator-sdk build quay.io/<your_user>/reverse-words-operator:latest --image-builder podman
podman push quay.io/<your_user>/reverse-words-operator:latest
~~~

## Generate the Cluster Service Version 

~~~sh
operator-sdk olm-catalog gen-csv --csv-version 0.1.0 --update-crds
~~~

https://operatorhub.io/preview