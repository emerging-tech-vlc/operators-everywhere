# Creating your very first Ansible Operator using the Operator Framework SDK

## Installing the Operator SDK

~~~sh
RELEASE_VERSION=v0.14.0
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
operator-sdk new reverse-words-operator --api-version=emergingtech.vlc/v1alpha1 --kind=ReverseWordsApp --type=ansible
cd reverse-words-operator
~~~

## Add the default properties

~~~sh
curl -Ls https://raw.githubusercontent.com/emerging-tech-vlc/operators-everywhere/master/ansible-operator/files/vars_main.yml -o roles/reversewordsapp/defaults/main.yml
~~~

## Add the controller logic

~~~sh
curl -Ls https://raw.githubusercontent.com/emerging-tech-vlc/operators-everywhere/master/ansible-operator/files/controller_main.yml -o roles/reversewordsapp/tasks/main.yml
~~~

## Build the Operator

~~~sh
export QUAY_USER=<your_quay_user>
operator-sdk build quay.io/${QUAY_USER}/reverse-words-operator-ansible:v0.1.0 --image-builder podman
podman push quay.io/${QUAY_USER}/reverse-words-operator-ansible:v0.1.0
~~~

## Deploy the Operator

~~~sh
# Update deployment manifests with the new image
sed -i "s|{{ REPLACE_IMAGE }}|quay.io/${QUAY_USER}/reverse-words-operator-ansible:v0.1.0|g" deploy/operator.yaml
sed -i 's|{{ pull_policy\|default('\''Always'\'') }}|Always|g' deploy/operator.yaml
# Load RBAC files into the cluster
oc create ns reverse-words-ansible
oc -n reverse-words-ansible create -f deploy/service_account.yaml
oc -n reverse-words-ansible create -f deploy/role.yaml
oc -n reverse-words-ansible create -f deploy/role_binding.yaml
# Load CRD into the cluster
oc create -f deploy/crds/emergingtech.vlc_reversewordsapps_crd.yaml
# Load Operator into the cluster
oc -n reverse-words-ansible create -f deploy/operator.yaml
~~~

## Load a CR to deploy a Reverse Word App Instance

~~~sh
cat <<EOF | oc -n reverse-words-ansible create -f -
apiVersion: emergingtech.vlc/v1alpha1
kind: ReverseWordsApp
metadata:
  name: example-reversewordsapp
spec:
  replicas: 3
  appVersion: "v0.0.3"
EOF
~~~
