"""Base class for all rule generators for Forseti scanners.

Derived classes can define _GetGlobalRules to specify rules which apply to
the global configuration (e.g. the organization), and _GetProjectRules to define
rules which apply to a specific project configuration.
"""

from __future__ import absolute_import
from __future__ import division
from __future__ import print_function

import abc


class BaseScannerRules(object):
  """Base class of all rule generators."""

  __metaclass__ = abc.ABCMeta

  @abc.abstractmethod
  def config_file_name(self):
    """Returns a string of the file name for this scanner's rule definitions."""
    pass

  def generate_rules(self, project_configs, global_config):
    """Generates rules dictionary for the given project and global configs."""
    # Get generic rules that apply to all projects.
    rules = self._get_global_rules(global_config)
    # Append project-specific rules.
    for project in project_configs:
      rules.extend(self._get_project_rules(project, global_config))
    return {'rules': rules}

  def _get_global_rules(self, global_config):
    """Get scanner rules that apply globally.

    Args:
      global_config (dict): A dictionary of global configuration values. This
          dictionary contains the keys: ['organization_id'].

    Returns:
      A list of scanner rules as dictionaries.
    """
    del global_config  # Unused.
    return []

  def _get_project_rules(self, project, global_config):
    """Get scanner rules that apply to the given project.

    Args:
      project (ProjectConfig): Configuration for a single project
      global_config (dict): A dictionary of global configuration values.

    Returns:
      A list of scanner rules as dictionaries.
    """
    del project, global_config  # Unused.
    return []
