# Kubernetes admission webhooks made easy in Javascript

Ever wanted to create you own admission webhook?
<br>
Don't have time to create your own webhook in golang, or python?

This project provides javascript admission webhooks to your kubernetes clusters:
- develop your admission rules in Javascript
- deploy your admission rules using Custom Resource Definitions
- choose between namespace and cluster scope using `JsAdmission` or `ClusterJsAdmission`

Adding custom webhooks is now as easy as adding a new object in Kubernetes:

```yaml
apiVersion: momiji.com/v1
kind: ClusterJsAdmission
metadata:
  name: sample-add-annotations
spec:
  kinds:
    - pods
  js: |
    function jsa_mutate(op, obj, sync, state) {
      if (op != "CREATE") return;
      if (obj.metadata.annotations == null)
        obj.metadata.annotations = {}
      obj.metadata.annotations["jsadmissions/sample-add-annotation"] = new Date().toISOString()
      return { Allowed: true, Result: obj }
    }
```

## Installation

### Clone the project

You need to clone the project or copy files from `kubernetes` folder.

```sh
$ git clone https://github.com/momiji/js-admissions-controller
$ cd js-admissions-controller
```

### Configuration

By default, the webhooks are deployed in the `kube-jsadmisions` namespace and only monitors `pods` creation.

To change this default behavior, simply update the yaml files according to your requirements:
- `kubernetes/crds.yaml` contains the two custom resource definitions JsAdmissions and ClusterJsAdmissions
- `kubernetes/deploy.yaml` contains the deployment for the web hooks controller
- `kubernetes/hooks.yaml` contains the default pod hooks for mutation and validation
- `kubernetes/namespace.yaml` contains the namespace definition, default is `kube-jsadmissions`
- `kubernetes/rbac.yaml` contains the RBAC to allow watching pods resources
- `kubernetes/install.sh` contains a simple install script

### Deploy the webhooks

Either install the objects manually one by one or simply use the provided `install.sh` script:

```sh
$ ./kubernetes/install.sh
```

### Deploy you first admission

```sh
$ kubectl apply -f - <<EOF
apiVersion: momiji.com/v1
kind: ClusterJsAdmission
metadata:
  name: sample-add-annotations
spec:
  kinds:
    - pods
  js: |
    function jsa_mutate(op, obj, sync, state) {
      if (op != "CREATE") return;
      if (obj.metadata.annotations == null)
        obj.metadata.annotations = {}
      obj.metadata.annotations["jsadmissions/sample-add-annotation"] = new Date().toISOString()
      return { Allowed: true, Result: obj }
    }
EOF
```

## A brief history

The idea for this project was born during the installation and configuration of SAS Viya4 for a customer.

In SAS Viya4, it is often necessary to type the nodes of the kubernetes cluster according to their role, and this is done by using the native taints and tolerations of kubernetes.

Unfortunately, taints are global to all pods on a node, which can impact other components than those deployed by SAS Viya4, like loggers or drivers, because it forbids them to start on tainted nodes if they don't have the appropriate tolerations.

To solve this problem, the first approach was to modify all the objects (pods, podtemplates, ...) generated in the installation phase (with kustomize), by updating their nodeAffinity/nodeSelector. But this requires to know in advance the exact list of all the objects that will be created by the operator in charge of the deployment.

An alternative idea then came up: make the modifications at the creation of the pods, by developing a mutating admission webhook, and to facilitate the development and testing of the 150 pods to be changed, the code must be located elsewhere than in the webhook and coded in a dynamic language like javascript. 

## Setup development environment

2 scenarios have been tested:
- k3s in docker, used to test the controller in a fresh new kubernetes node
- microk8s, used for development

> Using k3s requires some tricks to copy docker image into its internal registry, which is why it is only used for integration tests.

Requirements:
- ubuntu
- docker: `sudo apt install docker.io`
- kubectl: see https://kubernetes.io/fr/docs/tasks/tools/install-kubectl/
- jq: `sudo apt install jq`

Optional:
- microk8s with registry addon: `snap install microk8s ; microk8s enable registry`

### Using k3s

A full integration test can be performed on a docker k3s instance.
It takes around 1 min to finish on my 6 years old laptop.

```sh
$ ./tests/test-k3s.sh
```

### Using microk8s

Using microk8s is easier to use as its internal registry can be pushed from host using a simple `docker push` command.

To build and copy image into microk8s registry:

```sh
$ make
$ make docker
$ make local
```

To deploy:

```sh
$ ./tests/install.sh
$ kubectl wait deployment -n test-jsa test-jsa --for condition=Available=True --timeout=90s
```

To test:
```
$ ./tests/pods.sh
```

To update controller:

```sh
$ make local
$ kubectl rollout restart -n test-jsa deployment/test-jsa
$ kubectl wait deployment -n test-jsa test-jsa --for condition=Available=True --timeout=90s
```

To remove all objects except the CRD, simply delete `test-jsa` namespace:

```sh
$ kubectl delete namespace test-jsa
$ kubectl delete validatingwebhookconfigurations.admissionregistration.k8s.io test-jsa 
$ kubectl delete mutatingwebhookconfigurations.admissionregistration.k8s.io test-jsa 
```

## Javascript specification

### Managed functions

```text
// actions
function jsa_mutate(op, obj, [sync], [state]) -> { Allowed: bool, Message: str, Result: obj }
function jsa_validate(op, obj, [sync], [state]) -> { Allowed: bool, Message: str }

// init
function jsa_init([state])

// events
function jsa_created(obj, [sync], [state])
function jsa_updated(obj, old, [sync], [state])
function jsa_deleted(obj, [sync], [state])

// utils
function jsa_debug(s...)
function jsa_debugf(fmt, s...)
function jsa_log(s...)
function jsa_logf(fmt, s...)
```

### Function names

Methods managed by the runtime are all prefixed by `jsa_`.
<br>
Do not use `jsa_` in your custom functions to prevent future issues while upgrading.

A future release might implement a check to prevent using such prefix for user functions.

### Global variables

Do not use global variables to keep data accross function calls, as they are probably run from different runtime instances.

Use the `state` object for this, which is in read-only mode unless the `sync` parameter is also present in the function parameters, except for the `jsa_init(state)` function for which it is always synchronized.

### Known issues and solutions

The Javascript runtime included in the webhook is [dop251/goja](https://github.com/dop251/goja), which provides an incomplete javascript implementation.

There are some known issues with this runtime:
- ~~arrays: push() is not working~~

## Functions to implement

For parameters:
- names are case-sensitive
- names in brackets like `[sync]` or `[state]` are optional
- order is not important: `jsa_validate(obj,sync)` and `jsa_validate(sync,obj)` will both work correctly

Optional parameters:
- sync: when present, method is called synchronized, with value set to true
- state: state object that can be used to keep data

### jsa_mutate(op, obj, [sync]) -> { Allowed: bool, Message: str, Result: obj }

Parameters:
- op: operation, one of CREATE, UPDATE, DELETE
- obj: the object, like a Pod or a Deployment

Result:
- Allowed: boolean
- Message: error message, only used when Allowed if false
- Result: altered object, only used when Allowed is true

The mutation will fail only and only if:
- the return value is not null or undefined
- the return value contains a field Allowed
- the value of Allowed is exactly false

In all other case, the mutation will succeed:
- if the field Result is present and not null or undefined, the mutation is computed by comparing obj and Result
- otherwise, no patch is applied

Mutation and patch are logged when at least one Allowed is returned with a non-empty patch.

### jsa_validate(op, obj, [sync]) -> { Allowed: bool, Message: str }

Parameters:
- op: operation, one of CREATE, UPDATE, DELETE
- obj: the object, like a Pod or a Deployment

Return value:
- Allowed: boolean
- Message: error message, only used when Allowed if false
- Result: altered object, only used when Allowed is true

The validation will fail only and only if:
- the return value is not null or undefined
- the return value contains a field Allowed
- the value of Allowed is exactly false

In all other case, the validation will succeed.

Validation is logged when at least one Allowed is returned.

### jsa_init([state])

> There is no `sync` parameter as this method is always called synchronized.

### jsa_created(obj, [sync])

Parameters:
- obj: the object created, like a Pod or a Deployment

There is no return value for this function.

### jsa_updated(obj, old, [sync])

Parameters:
- oldObj: the object before update, like a Pod or a Deployment
- newObj: the object after update, like a Pod or a Deployment

There is no return value for this function.

> Remember that updates are sent each time the resourceVersion field of the object is changed.
> This can happen when object is patched, but also when object status changes.

### jsa_deleted(obj, [sync])

Parameters:
- obj: the object created, like a Pod or a Deployment

## Javascript utilities

TODO

## Examples

### Adding a new annotation to all pods

In this example we want to add a new annotation `jsadmissions/date` with the current date to all pods.

Here, we simply need to:
- implement `jsa_mutate` function to update the object

```js
function jsa_mutate(op, obj) {
    if (op != "CREATE" || obj.kind !== "Pod") return;
    if (obj.metadata.annotations == null)
        obj.metadata.annotations = {};
    obj.metadata.annotations["jsadmissions/date"] = new Date().toISOString();
    return { Allowed: true, Result: obj };
}
```

### Limit the number of pods accross default-* namespaces

In this example we want to count the number of pods created across multiple namespaces, to prevent going above a limit of 40 pods.

Here, we would need to:
- implement `jsa_init` function to initialize the value to 0
- implement `jsa_created` and `jsa_deleted` functions to update the value
- implement `jsa_validate` to test and eventually prevent object creation when limit is reached
- use `state` to store the number of existing pods
- use `sync` to be able to read/write this value atomically

```js
// entrypoints
function jsa_init(state) {
    state.podCount = 0;
}
function jsa_created(obj, sync, state) {
    // Check object kind and namespace
    if (!check(obj)) return;
    // Update state
    state.podCount++;
}
function jsa_deleted(obj, sync, state) {
    // Check object kind and namespace
    if (!check(obj)) return;
    // Update state
    state.podCount--;
}
function jsa_validate(op, obj, sync, state) {
    // Check object kind and namespace
    if (op != "CREATE" || !check(obj)) return;
    // Check pod count < limit
    if (state.podCount >= POD_LIMIT) {
        return { Allowed: false, Message: "Max number of pods has been reached" };
    }
    return;
}

// custom code
var NAMESPACE_REGEX = /^default($|-)/;
var POD_LIMIT = 40;
function check(obj) {
    return obj.kind === "Pod" && obj.metadata.namespace.match(NAMESPACE_REGEX) != null;
}
```

## Real world use case

This example is taken from our SAS Viya4 installation process, where we need to replace taints/tolerations logic by a nodeSelector field, with additional custom hacks for cas workers and crunchy database.

```js
/*
# update nodeSelector for all pods
# - if casoperator.sas.com/node-type=worker => 'cas'
# - if casoperator.sas.com/node-type=* => 'compute'
# - if launcher.sas.com/job-type=compute-server => 'compute'
# - if workload.sas.com/class => value
# - else => stateless
# update resources for cas workers
# - if casoperator.sas.com/node-type=worker => use dedicated annotations
# update database command for crunchy
# - if postgres-operator.crunchydata.com/data=postgres => update command of container database
*/
function container_by_name(containers, name) {
    for (var c of containers) {
        if (c.name == name) return c;
    }
    return null;
}
function jsa_mutate(obj) {
    // init labels
    if (obj.metadata.labels == null)
        obj.metadata.labels = {};
    // compute new class
    var workload = "stateless";
    var labels = obj.metadata.labels;
    if (labels["casoperator.sas.com/node-type"] != null) {
        var v = labels["casoperator.sas.com/node-type"];
        if (v == "worker") {
            workload = "cas";
            // update resources for cas workers
            var container = container_by_name(obj.spec.containers, "sas-cas-server");
            var annos = obj.metadata.annotations;
            if (container != null) {
                if (annos["casworker.sas.com/cpu-limit"] != null)
                    container.resources.limits.cpu = annos["casworker.sas.com/cpu-limit"];
                if (annos["casworker.sas.com/cpu-request"] != null)
                    container.resources.requests.cpu = annos["casworker.sas.com/cpu-request"];
                if (annos["casworker.sas.com/mem-limit"] != null)
                    container.resources.limits.memory = annos["casworker.sas.com/mem-limit"];
                if (annos["casworker.sas.com/mem-request"] != null)
                    container.resources.requests.memory = annos["casworker.sas.com/mem-request"];
            }
        } else {
            workload = "compute";
        }
    } else if (labels["launcher.sas.com/job-type"] == "compute-server") {
        workload = "compute";
    } else if (labels["workload.sas.com/class"] != null) {
        workload = labels["workload.sas.com/class"];
    }
    // update database command for crunchy
    if (labels["postgres-operator.crunchydata.com/data"] == "postgres") {
        var container = container_by_name(obj.spec.containers, "database");
        if (container != null && container.command != null && container.command.join(" ") == "patroni /etc/patroni") {
            container.command = [ "/bin/bash", "-ceEx", "ulimit -c 0 ; ulimit -a ; patroni /etc/patroni" ];
        } else {
            return { Allowed: false, Message: "Invalid postgres container database command, fix code or make it more generic" };
        }
    }
    // add label with new class
    labels["workload.sas.com/jsa-class"] = workload;
    // add nodeSelector
    if (obj.spec.nodeSelector == null)
        obj.spec.nodeSelector = {};
    obj.spec.nodeSelector["workload.sas.com/" + workload] = "yes";
    // return success
    return { Allowed: true, Result: obj };
}
```

## Notes

### Admissions execution order

By design, namespace admissions are executed **before** cluster admissions, in name order.
This way, cluster mutations have higher priority than namespace admissions.

However, if you have security concerns, the good practice is to implement validations in addition to mutations.

### Webhooks configuration

There should be no reason to have more than one webhook for namespaces and for clustered admissions.
Doing so may result in admissions been executed several times, which should not be what is expected.

### Limit admissions kinds

If you need to prevent namespace admissions to mutate/validate some resources,
you might want to add cluster admissions to validate the creation and modification of JsAdmissions.
