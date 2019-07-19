"""Test rule that fails if a project config is not valid."""

def _impl(ctx):
    """Core implementation of config_test rule."""
    ctx.actions.write(
        content = "{config_loader} --config_path {path}".format(
            config_loader = ctx.executable._config_loader.short_path,
            path = ctx.file.config.short_path,
        ),
        output = ctx.outputs.executable,
        is_executable = True,
    )

    runfiles = ctx.runfiles(files = [
        ctx.file._config_loader,
        ctx.file._schema,
        ctx.file.config,
    ] + ctx.files.imports)
    return [DefaultInfo(runfiles = runfiles)]

config_test = rule(
    attrs = {
        "config": attr.label(
            mandatory = True,
            doc = "The project config to validate.",
            allow_single_file = True,
        ),
        "imports": attr.label_list(
            allow_files = True,
            allow_empty = True,
            doc = "Additional configs or templates to import.",
        ),
        "_config_loader": attr.label(
            default = Label("//deploy/cmd/load_config"),
            doc = "The config loader binary. Internal attribute and should not be set by users.",
            cfg = "host",
            executable = True,
            allow_single_file = True,
        ),
        # This attribute is added purely for the purpose of making the schema file available
        # in the runfiles tree.
        "_schema": attr.label(
            default = Label("//deploy:project_config.yaml.schema"),
            doc = "The config schema. Internal attribute and should not be set by users.",
            cfg = "host",
            allow_single_file = True,
        ),
    },
    test = True,
    implementation = _impl,
)
