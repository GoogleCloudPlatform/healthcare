"""Rule Generator for Forseti's Log Sink Scanner.

Creates global rules to require and only allow a BigQuery log sink in all
projects.

Creates project-specific rules to require and whitelist the single, expected
Log Sink to the configured BigQuery destination.
"""

from __future__ import absolute_import
from __future__ import division
from __future__ import print_function

from deploy.rule_generator.scanners import base_scanner_rules

# TODO: Change the filter to be specifically an audit logs sink once
# deployed log sinks use that filter.
_SINK_RULE_FILTER = '*'


def _make_rule(name, mode, resources, destination):
  """Helper function to build a rule dictionary."""
  for resource in resources:
    resource['applies_to'] = ('self'
                              if resource['type'] == 'project' else 'children')
  return {
      'name': name,
      'mode': mode,
      'resource': resources,
      'sink': {
          'destination': destination,
          'filter': _SINK_RULE_FILTER,
          'include_children': '*',
      },
  }


class LogSinkScannerRules(base_scanner_rules.BaseScannerRules):
  """Scanner rule generator for the Stackdriver Log Sink scanner."""

  def config_file_name(self):
    return 'log_sink_rules.yaml'

  def _get_global_rules(self, global_config, project_configs):
    resources = self._get_resources(global_config, project_configs)
    destination = 'bigquery.googleapis.com/*'
    # The two rules differ only in name and mode.
    return [
        _make_rule(
            name='Require a BigQuery Log sink in all projects.',
            mode='required',
            resources=resources,
            destination=destination),
        _make_rule(
            name='Only allow BigQuery Log sinks in all projects.',
            mode='whitelist',
            resources=resources,
            destination=destination),
    ]

  def _get_project_rules(self, project, global_config):
    del global_config  # Unusued.
    # Generate a narrower pair of rules that require and only allow a specific
    # log sink destination.
    project_id = project.project_id
    destination = project.get_audit_log_sink_destination()
    resources = [{'type': 'project', 'resource_ids': [project_id]}]

    return [
        _make_rule(
            name='Require Log sink for project {}.'.format(project_id),
            mode='required',
            resources=resources,
            destination=destination),
        _make_rule(
            name='Whitelist Log sink for project {}.'.format(project_id),
            mode='whitelist',
            resources=resources,
            destination=destination),
    ]
