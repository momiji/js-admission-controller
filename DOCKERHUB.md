Kubernetes admission webhook controller with javascript based admissions defined in custom resources

# Quick reference

- **Maintained by**:
  <br>[Christian Bourgeois](https://github.com/momiji/js-admissions-controller) (the developer)


- **Where to get help**:
  <br>[Documentation](https://github.com/momiji/js-admissions-controller/blob/main/README.md) (GitHub)
  <br>[Issues](https://github.com/momiji/js-admissions-controller/issues) (GitHub)

# Tags

Releases follow semver specification
- **x.y.z** is the full version number
- **x.y** is the last x.y.z version
- **x** is the last x.y.z version

Additional tags:
- **edge**: main version always up-to-date, unstable and that may not work at all
- **latest**: last stable version

# What is it?

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

View more in the [documentation](https://github.com/momiji/js-admissions-controller/blob/main/README.md).
