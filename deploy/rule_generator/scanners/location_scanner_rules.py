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

  def _get_project_rules(self, project_config, global_config):
    """Gets project specific location rules.

    A rule is created from the buckets specified in the project config. For each
    resource location one rule is created. There are also rules created for
    the audit log resources.

    Args:
      project_config (ProjectConfig): project config to build rules from.
      global_config (dict): global config to build rules from.

    Returns:
      List[dict] - The rules dictionaries.
    """
    # TODO: add some global whitelists based on the locations of
    # per-project resources.
    rules = []

    loc_to_resource_map = collections.defaultdict(
        lambda: collections.defaultdict(list)
    )

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

    for loc in locs:
      resource_map = loc_to_resource_map[loc]
      applies_to = [
          {
              'type': res_type,
              'resource_ids': res_ids,
          }
          for res_type, res_ids in resource_map.items()
      ]

      rules.append({
          'name': 'Project {} resource whitelist for location {}.'.format(
              project_config.project_id, loc),
          'mode': 'whitelist',
          'resource': [{
              'type': 'project',
              'resource_ids': [project_config.project_id],
          }],
          'applies_to': applies_to,
          'locations': [loc],
      })

    audit_log_bucket = project_config.get_audit_log_bucket()
    if audit_log_bucket:
      rules.append({
          'name': 'Project {} audit logs bucket location whitelist.'.format(
              project_config.project_id),
          'mode': 'whitelist',
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
      rules.append({
          'name': 'Project {} audit logs dataset location whitelist.'.format(
              project_config.project_id),
          'mode': 'whitelist',
          'resource': [{
              'type': 'project',
              'resource_ids': [project_config.audit_logs_project_id],
          }],
          'applies_to': [{
              'type': 'dataset',
              'resource_ids': ['{}:{}'.format(
                  project_config.audit_logs_project_id,
                  project_config.audit_logs_bigquery_dataset['name'],
              )],
          }],
          'locations': [project_config.audit_logs_bigquery_dataset['location']],
      })
    return rules
