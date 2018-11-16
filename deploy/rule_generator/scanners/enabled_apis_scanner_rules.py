"""Rule Generator for Forseti's Enabled APIs Scanner.

Produces whitelists for APIs defined in the project configs as well as a
global whitelist to encompass all APIs defined.
"""

from __future__ import absolute_import
from __future__ import division
from __future__ import print_function

from deploy.rule_generator.scanners import base_scanner_rules


def _create_whitelist_rule(name, project_id, apis):
  """Creates a whitelist rule for the project and apis."""
  return {
      'name': name,
      'mode': 'whitelist',
      'resource': [{
          'type': 'project',
          'resource_ids': [project_id],
      }],
      'services': apis,
  }


class EnabledApisScannerRules(base_scanner_rules.BaseScannerRules):
  """Scanner rule generator for the Enabled APIs scanner."""

  def config_file_name(self):
    return 'enabled_apis_rules.yaml'

  def _get_global_rules(self, global_config):
    apis = global_config.get('allowed_apis', [])
    if not apis:
      return []
    return [_create_whitelist_rule('Global API whitelist.', '*', apis)]

  def _get_project_rules(self, project, global_config):
    del global_config  # Unusued.
    # If project config specifies APIs, created a more restricted whitelist.
    if not project.enabled_apis:
      return []
    return [_create_whitelist_rule(
        'API whitelist for {}'.format(project.project_id), project.project_id,
        project.enabled_apis)]

