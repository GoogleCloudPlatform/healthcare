# GxP-Aligned Life Sciences R&D Platform DPT Templates

**Status:** Stable

**Solution Guide:**
[Setting up a GxP-Aligned Life Sciences Platform using Data Protection Toolkit](http://services.google.com/fh/files/misc/gxp_technical_solution_guide.pdf)

The essential templates required to deploy the GxP-Aligned Life Sciences R&D
Platform are:

- `variables.yaml`
- `config.yaml`

## Deploying the templates

`config.yaml` file is imported by `variables.yaml` file. So the entry point
would be `variables.yaml` file.

```shell
  bazel run //cmd/apply:apply -- --config_path=./variables.yaml
```

The Cloud Function script used in the `google_cloudfunctions_function` block can
be found
[here](https://cloud.google.com/solutions/automating-classification-of-data-uploaded-to-cloud-storage).

The setup steps for the DPT are clearly mentioned in the official
[DPT repo](https://github.com/GoogleCloudPlatform/healthcare/tree/master/deploy).
