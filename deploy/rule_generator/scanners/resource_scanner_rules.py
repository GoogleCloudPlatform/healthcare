"""Rule Generator for Forseti's Resource Scanner.

Creates a global rule containing resource trees for each project defined in the
config.
"""

from __future__ import absolute_import
from __future__ import division
from __future__ import print_function

from deploy.rule_generator.scanners import base_scanner_rules

_APPLICABLE_RESOURCE_TYPES = ['project', 'bucket', 'dataset', 'instance']


class ResourceScannerRules(base_scanner_rules.BaseScannerRules):
  """Scanner rule generator for the Resource scanner."""

  def config_file_name(self):
    return 'resource_rules.yaml'

  def _get_global_rules(self, global_config, project_configs):
    return [{
        'name': 'Project resource trees.',
        'mode': 'required',
        'resource_types': _APPLICABLE_RESOURCE_TYPES,
        'resource_trees': _get_resource_trees(project_configs),
    }]


def _get_resource_trees(project_configs):
  """Get the resource trees for the given project configs."""

  # Ignore unmonitored projects.
  trees = [{'type': 'project', 'resource_id': '*'}]

  for project_config in project_configs:
    children = [{
        'type': 'bucket',
        'resource_id': bucket.id,
    } for bucket in project_config.get_buckets()]

    children.extend({
        'type':
            'dataset',
        'resource_id':
            '{}:{}'.format(project_config.project_id, dataset['name'])
    } for dataset in project_config.bigquery_datasets)

    children.extend({
        'type': 'instance',
        'resource_id': gce_instance.id,
    } for gce_instance in project_config.get_gce_instances())

    project_tree = {
        'type': 'project',
        'resource_id': project_config.project_id,
    }
    if children:
      project_tree['children'] = children
    trees.append(project_tree)

  return trees
