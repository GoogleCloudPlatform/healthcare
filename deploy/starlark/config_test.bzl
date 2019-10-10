"""Test rule that fails if a project config is not valid."""

def _impl(ctx):
    """Core implementation of _config_test rule."""
    content = "{config_loader} --config_path {path}".format(
        config_loader = ctx.executable._config_loader.short_path,
        path = ctx.file.config.short_path,
    )

    ctx.actions.write(
        content = content,
        output = ctx.outputs.executable,
        is_executable = True,
    )

    runfiles = [
        ctx.file._config_loader,
        ctx.file._generated_fields_schema,
        ctx.file._project_config_schema,
        ctx.file.config,
    ] + ctx.files.deps

    return [DefaultInfo(runfiles = ctx.runfiles(files = runfiles))]

_config_test = rule(
    attrs = {
        "config": attr.label(
            mandatory = True,
            doc = "The project config to validate.",
            allow_single_file = True,
        ),
        "deps": attr.label_list(
            allow_files = True,
            doc = "Additional dependent configs, templates or generated fields file to import.",
        ),
        "_config_loader": attr.label(
            default = Label("//cmd/load_config"),
            doc = "The config loader binary. Internal attribute and should not be set by users.",
            cfg = "host",
            executable = True,
            allow_single_file = True,
        ),
        # The following attributes are added purely for the purpose of making
        # schema files available in the runfiles tree.
        "_generated_fields_schema": attr.label(
            default = Label("//:generated_fields.yaml.schema"),
            doc = "The generated fields schema. Internal attribute and should not be set by users.",
            cfg = "host",
            allow_single_file = True,
        ),
        "_project_config_schema": attr.label(
            default = Label("//:project_config.yaml.schema"),
            doc = "The project config schema. Internal attribute and should not be set by users.",
            cfg = "host",
            allow_single_file = True,
        ),
    },
    test = True,
    implementation = _impl,
)

def config_test(**kwargs):
    """Test rule that fails if a project config is not valid.

    Args:
      **kwargs: Same attrs of _config_test.
    """
    if "deps" in kwargs and kwargs["deps"] == []:
        fail("`deps` is specified but resolved to no file. It might be because your glob() pattern contains typos.")

    _config_test(**kwargs)
