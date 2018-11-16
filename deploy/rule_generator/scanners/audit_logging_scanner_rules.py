"""Rule Generator for Forseti's Audit Logging Scanner.

Produces a rule requiring audit logs for all projects.
"""

from __future__ import absolute_import
from __future__ import division
from __future__ import print_function

from deploy.rule_generator.scanners import base_scanner_rules

# Rule requiring audit logs for all projects.
_GLOBAL_AUDIT_LOGS_RULE = {
    'name': 'Require all Cloud Audit logs.',
    'resource': [{
        'type': 'project',
        'resource_ids': ['*'],
    }],
    'service': 'allServices',
    'log_types': [
        'ADMIN_READ',
        'DATA_READ',
        'DATA_WRITE',
    ]
}


class AuditLoggingScannerRules(base_scanner_rules.BaseScannerRules):
  """Scanner rule generator for the Audit Logging scanner."""

  def config_file_name(self):
    return 'audit_logging_rules.yaml'

  def _get_global_rules(self, global_config):
    del global_config  # Unused.
    # The Audit Logging scanner only requires a single, global rule.
    return [_GLOBAL_AUDIT_LOGS_RULE]
