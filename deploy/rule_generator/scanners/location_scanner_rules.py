"""Rule Generator for Forseti's Location Scanner.

Creates rules to ensure GCP resources are located in the regions they were
configured to be in.
"""

from __future__ import absolute_import
from __future__ import division
from __future__ import print_function

import collections

from deploy.rule_generator.scanners import base_scanner_rules


class LocationScannerRules(base_scanner_rules.BaseScannerRules):
  """Scanner rule generator for the Lien scanner."""

  def config_file_name(self):
    return 'location_rules.yaml'

  def generate_rules(self, project_configs, global_config):
    """Gets project specific location rules.

    A location whitelist is created for each location set for a resource.
    The locations are joined to form a single global whitelist rule as well.

    Args:
      project_configs (List[ProjectConfig]): project config to build rules from.
      global_config (dict): global config to build rules from.

    Returns:
      List[dict] - The rules dictionaries.
    """
    project_rules = []

    all_locs = set()

    for project_config in project_configs:
      loc_to_resource_map = collections.defaultdict(
          lambda: collections.defaultdict(list))

      for bucket in project_config.get_buckets():
        loc = bucket.location.upper()
        loc_to_resource_map[loc]['bucket'].append(bucket.id)

      for dataset in project_config.bigquery_datasets:
        loc = dataset['location'].upper()
        dataset_id = '{}:{}'.format(project_config.project_id, dataset['name'])
        loc_to_resource_map[loc]['dataset'].append(dataset_id)

      for gce_instance in project_config.get_gce_instances():
        loc = gce_instance.location.upper()
        loc_to_resource_map[loc]['instance'].append(gce_instance.id)

      locs = sorted(loc_to_resource_map.keys())
      all_locs.update(locs)

      for loc in locs:
        resource_map = loc_to_resource_map[loc]
        applies_to = [{
            'type': res_type,
            'resource_ids': res_ids,
        } for res_type, res_ids in resource_map.items()]

        project_rules.append({
            'name':
                'Project {} resource whitelist for location {}.'.format(
                    project_config.project_id, loc),
            'mode':
                'whitelist',
            'resource': [{
                'type': 'project',
                'resource_ids': [project_config.project_id],
            }],
            'applies_to':
                applies_to,
            'locations': [loc],
        })

      audit_log_bucket = project_config.get_audit_log_bucket()
      if audit_log_bucket:
        project_rules.append({
            'name':
                'Project {} audit logs bucket location whitelist.'.format(
                    project_config.project_id),
            'mode':
                'whitelist',
            'resource': [{
                'type': 'project',
                'resource_ids': [project_config.audit_logs_project_id],
            }],
            'applies_to': [{
                'type': 'bucket',
                'resource_ids': [audit_log_bucket.id],
            }],
            'locations': [audit_log_bucket.location],
        })

      if project_config.audit_logs_bigquery_dataset:
        project_rules.append({
            'name':
                'Project {} audit logs dataset location whitelist.'.format(
                    project_config.project_id),
            'mode':
                'whitelist',
            'resource': [{
                'type': 'project',
                'resource_ids': [project_config.audit_logs_project_id],
            }],
            'applies_to': [{
                'type':
                    'dataset',
                'resource_ids': [
                    '{}:{}'.format(
                        project_config.audit_logs_project_id,
                        project_config.audit_logs_bigquery_dataset['name'],
                    )
                ],
            }],
            'locations': [
                project_config.audit_logs_bigquery_dataset['location']
            ],
        })

    global_rule = {
        'name':
            'Global location whitelist.',
        'mode':
            'whitelist',
        'resource': [{
            'type': 'organization',
            'resource_ids': [global_config['organization_id']],
        }],
        'applies_to': [{
            'type': '*',
            'resource_ids': ['*'],
        }],
        'locations':
            sorted(list(all_locs)),
    }
    return {'rules': [global_rule] + project_rules}
