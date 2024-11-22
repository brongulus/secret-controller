# secret-controller
The controller works by adding immutability to all the secrets that are currently being used by pods however it restricts it to a set of images that are specified by the ImmutableImages custom resource.

Upon any updates to to ImmutableImages or creation of a new pod, the reconciliation occurs, it looks for all the new secrets that should be marked as immutable by looking for the various ways in which a secret is attached to containers.

Preventing changes to the data of an existing Secret has the following benefits:
- protects you from accidental (or unwanted) updates that could cause applications outages
- improves cluster performance by reducing apiserver 

Ref: [Secrets](https://kubernetes.io/docs/concepts/configuration/secret/#secret-immutable)

## TODOs 
- Check if it's possible to edit the secret from the pod itself!
- Remove statefulness from the CR to allow for updates to the list
Think about doing it without the map somehow (if the status thing happens, what we can do is every reconcile, create the spec and status, so that there's no state to keep track of)
- Check when secret is deleted and a pod is created that refers it (secret Get failure)
- Add namespace to the CR as well
- Validating Webhook (list of immutable secrets in CR status, on removal of image from the list, update the secret list, annotation?), webhook implements the immutability indirectly to not have to resort to deletion of secrets and pods.

## Getting Started

### Prerequisites
- go version v1.22.0+
- docker version 17.03+.
- kubectl version v1.11.3+.
- Access to a Kubernetes v1.11.3+ cluster.

### To Deploy on the cluster
**Build and push your image to the location specified by `IMG`:**

```sh
make docker-build docker-push IMG=<some-registry>/secret-controller:tag
```

**NOTE:** This image ought to be published in the personal registry you specified.
And it is required to have access to pull the image from the working environment.
Make sure you have the proper permission to the registry if the above commands don’t work.

**Install the CRDs into the cluster:**

```sh
make install
```

**Deploy the Manager to the cluster with the image specified by `IMG`:**

```sh
make deploy IMG=<some-registry>/secret-controller:tag
```

> **NOTE**: If you encounter RBAC errors, you may need to grant yourself cluster-admin
privileges or be logged in as admin.

**Create instances of your solution**
You can apply the samples (examples) from the config/sample:

```sh
kubectl apply -k config/samples/
```

>**NOTE**: Ensure that the samples has default values to test it out.

### To Uninstall
**Delete the instances (CRs) from the cluster:**

```sh
kubectl delete -k config/samples/
```

**Delete the APIs(CRDs) from the cluster:**

```sh
make uninstall
```

**UnDeploy the controller from the cluster:**

```sh
make undeploy
```

## Project Distribution

Following are the steps to build the installer and distribute this project to users.

1. Build the installer for the image built and published in the registry:

```sh
make build-installer IMG=<some-registry>/secret-controller:tag
```

NOTE: The makefile target mentioned above generates an 'install.yaml'
file in the dist directory. This file contains all the resources built
with Kustomize, which are necessary to install this project without
its dependencies.

2. Using the installer

Users can just run kubectl apply -f <URL for YAML BUNDLE> to install the project, i.e.:

```sh
kubectl apply -f https://raw.githubusercontent.com/<org>/secret-controller/<tag or branch>/dist/install.yaml
```

## Contributing
// TODO(user): Add detailed information on how you would like others to contribute to this project

**NOTE:** Run `make help` for more information on all potential `make` targets

More information can be found via the [Kubebuilder Documentation](https://book.kubebuilder.io/introduction.html)

## License

Copyright 2024.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

