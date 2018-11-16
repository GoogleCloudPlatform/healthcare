"""Rule Generator for Forseti's Big Query Scanner.

Produces global rules blacklisting public and domain-level dataset access.
"""

from __future__ import absolute_import
from __future__ import division
from __future__ import print_function

from deploy.rule_generator.scanners import base_scanner_rules


class BigQueryScannerRules(base_scanner_rules.BaseScannerRules):
  """Scanner rule generator for the BigQuery ACL scanner."""

  def config_file_name(self):
    return 'bigquery_rules.yaml'

  def _get_global_rules(self, global_config):
    """Overrides base_scanner_rules.BaseScannerRules._get_global_rules."""
    return [{
        'name': 'No public, domain or special group dataset access.',
        'mode': 'blacklist',
        'resource': [{
            'type': 'organization',
            'resource_ids': [global_config['organization_id']],
        }],
        'dataset_ids': ['*'],
        'bindings': [{
            'role': '*',
            'members': [
                {'domain': '*'},
                {'special_group': '*'},
            ],
        }],
    }]

  def _get_project_rules(self, project, global_config):
    """Overrides base_scanner_rules.BaseScannerRules._get_project_rules."""
    del global_config  # Unused.

    rules = []

    dataset_rule_num = 1
    for dataset_ids, bindings in project.get_project_bigquery_bindings():
      name = 'Whitelist for dataset(s): {}'.format(', '.join(dataset_ids))
      if len(name) > 127:
        name = name[:124] + '...'
      rules.append({
          'name': name,
          'mode': 'whitelist',
          'dataset_ids': dataset_ids,
          'resource': [{
              'type': 'project',
              'resource_ids': [project.project_id],
          }],
          'bindings': bindings,
      })
      dataset_rule_num += 1

    audit_logs_project_rule = {
        'name': 'Whitelist for project %s audit logs' % project.project_id,
        'mode': 'whitelist',
        'dataset_ids': [
            '%s:%s' % (project.audit_logs_project_id,
                       project.audit_logs_bigquery_dataset['name']),
        ],
        'resource': [{
            'type': 'project',
            'resource_ids': [project.audit_logs_project_id],
        }],
        'bindings': project.get_audit_logs_bigquery_bindings(),
    }
    rules.append(audit_logs_project_rule)

    return rules
