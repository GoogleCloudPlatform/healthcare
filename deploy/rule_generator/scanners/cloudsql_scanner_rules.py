"""Rule Generator for Forseti's Cloud SQL Scanner.

Produces global rules blacklisting publicly exposed Cloud SQL instances.
"""

from __future__ import absolute_import
from __future__ import division
from __future__ import print_function

from deploy.rule_generator.scanners import base_scanner_rules


class CloudSqlScannerRules(base_scanner_rules.BaseScannerRules):
  """Scanner rule generator for the Cloud SQL scanner."""

  def config_file_name(self):
    return 'cloudsql_rules.yaml'

  def _get_global_rules(self, global_config, project_configs):
    rules = []
    for ssl_enabled in [False, True]:
      name = 'Disallow publicly exposed cloudsql instances (SSL {}).'.format(
          'enabled' if ssl_enabled else 'disabled')
      rules.append({
          'name': name,
          'resource': self._get_resources(global_config, project_configs),
          'instance_name': '*',
          'authorized_networks': '0.0.0.0/0',
          'ssl_enabled': str(ssl_enabled),
      })
    return rules
