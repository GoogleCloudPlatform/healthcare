# Environments Config Example

NOTE: Before this sample, please see the full example to understand how a single
complete environment can be setup.

This directory contains an example on creating multiple environments to create
multiple instances of the same project. In this example, we will create a dev
and prod instance of the project. Because the same project template file is
imported in both, they will be deployed in the same way.

If using Terraform directly, this is difficult to achieve. While the DPT config
ensures the projects are deployed with the same inputs, if using Terraform you
will likely need to create a module for each shared resource and then reference
the module in all configs.
