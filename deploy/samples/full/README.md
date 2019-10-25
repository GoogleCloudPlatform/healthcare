# Full Config Example

This directory contains a complete example to setup multiple GCP projects within
an org which can host sensitive data.

A similar pattern can be found under
[Cloud Foundation Fabric](https://github.com/terraform-google-modules/cloud-foundation-fabric/tree/master/foundations).

Through DPT, you can configure and deploy the fabric recommended architecture.


## Deployment steps

You would need to make a copy of the configs in the samples in your own version
control system, but the commands here will be specific for the projects deployed
in these configs.

To deploy, the maintainers of the shared config (typically a devops or
data compliance team) should first deploy the shared projects:

```
$ bazel run cmd/apply:apply -- \
  --config_path=samples/full/shared.yaml \
```

Then, `team1` can deploy their subprojects:

```
$ bazel run cmd/apply:apply -- \
  --config_path=samples/full/team1/config.yaml
  --projects=example-analysis
```

And `team2` can also deploy their subprojects independently:

```
$ bazel run cmd/apply:apply -- \
  --config_path=samples/full/team2/config.yaml
  --projects=example-data-123,example-data-456
```

NOTE: All subteams must have owners access to the central audit and devops
projects for deployment to be able to create resources within them.
