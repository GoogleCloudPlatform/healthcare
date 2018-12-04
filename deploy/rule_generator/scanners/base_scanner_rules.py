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
    rules = self._get_global_rules(global_config, project_configs)
    # Append project-specific rules.
    for project in project_configs:
      rules.extend(self._get_project_rules(project, global_config))
    return {'rules': rules}

  def _get_global_rules(self, global_config, project_configs):
    """Get scanner rules that apply globally.

    Args:
      global_config (dict): A dictionary of global configuration values.
      project_configs (List[ProjectConfig]): All configurations for projects.

    Returns:
      A list of scanner rules as dictionaries.
    """
    del global_config, project_configs  # Unused.
    return []

  def _get_project_rules(self, project, global_config):
    """Get scanner rules that apply to the given project.

    Args:
      project (ProjectConfig): Configuration for a single project.
      global_config (dict): A dictionary of global configuration values.

    Returns:
      A list of scanner rules as dictionaries.
    """
    del project, global_config  # Unused.
    return []

  @classmethod
  def _get_resources(cls, global_config, project_configs):
    """Get the resources list for a Forseti rule based on the given configs."""
    if 'organization_id' in global_config:
      return [{
          'type': 'organization',
          'resource_ids': [global_config['organization_id']],
      }]
    elif 'folder_id' in global_config:
      return [{
          'type': 'folder',
          'resource_ids': [global_config['folder_id']],
      }]
    else:
      return [{
          'type': 'project',
          'resource_ids': [conf.project_id for conf in project_configs]
      }]
