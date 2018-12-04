"""Rule Generator for Forseti's IAM Scanner.

Creates global rules to require a domain group as owner and a blacklist to
prevent user members in project-level roles. Also creates a global bucket
whitelist, restricting to Google groups, users and service accounts.

Project-specific whitelists are created for project-level and bucket roles.
"""

from __future__ import absolute_import
from __future__ import division
from __future__ import print_function

import re

from deploy.rule_generator.scanners import base_scanner_rules

# Empty whitelists aren't supported, so use this entry for whitelists that will
# match to nobody.
_NOBODY = 'user:nobody'

# Format lists of strings to signify what default members are allowed for
# projects and buckets. The field 'domain' will be passed in from the global
# config.
_ALLOWED_PROJECT_MEMBER_FMTS = ['group:*@{domain}',
                                'serviceAccount:*.gserviceaccount.com']
_ALLOWED_BUCKET_MEMBER_FMTS = _ALLOWED_PROJECT_MEMBER_FMTS + ['user:*@{domain}']


class IamScannerRules(base_scanner_rules.BaseScannerRules):
  """Scanner rule generator for the IAM scanner."""

  def config_file_name(self):
    return 'iam_rules.yaml'

  def generate_rules(self, project_configs, global_config):
    """Generates rule dictionaries from the given configs.

    Overrides BaseScannerRules.generate_rules.

    Args:
      project_configs (List[ProjectConfig]): project configs to build rules
        from.
      global_config (dict): global config to build rules from.

    Returns:
      List[dict]: Rule dictionaries. Each item in the list is one rule. There
        will be global rules applicable to all projects if there is a domain set
        as well as one rule for each project in the project configs.
    """
    project_rules = []

    project_bindings = []
    bucket_bindings = []

    for project in project_configs:
      # Append project-specific rules.
      project_rules.extend(self._get_project_iam_rules(project))

      project_bindings.append(project.get_project_bindings())
      bucket_bindings.extend(
          bindings for _, bindings in project.get_bucket_bindings())

    domain = global_config.get('domain')
    if not domain:
      return {'rules': project_rules}  # No global rules if domain is not set.

    global_rules = [
        {
            'name':
                'All projects must have an owner group from the domain',
            'mode':
                'required',
            'resource': [{
                'type': 'project',
                'applies_to': 'self',
                'resource_ids': ['*'],
            }],
            'inherit_from_parents':
                True,
            'bindings': [{
                'role': 'roles/owner',
                'members': ['group:*@' + domain],
            }],
        },
    ]

    global_rules.append(
        _get_global_whitelist_rule('project', _ALLOWED_PROJECT_MEMBER_FMTS,
                                   project_bindings, domain))
    global_rules.append(
        _get_global_whitelist_rule('bucket', _ALLOWED_BUCKET_MEMBER_FMTS,
                                   bucket_bindings, domain))

    return {'rules': global_rules + project_rules}

  def _get_project_iam_rules(self, project):
    # Generate a narrower whitelist for each project and bucket. These rules
    # could also be duplicated as an additional 'required' rule to enforce an
    # exact match.
    rules = []

    # Project-level IAM roles:
    project_bindings = project.get_project_bindings()
    rules.append(_get_project_rule(
        'Role whitelist for project {}'.format(project.project_id),
        'whitelist', 'project', [project.project_id], project_bindings))

    for bucket_ids, bindings in project.get_bucket_bindings():
      name = 'Role whitelist {} for buckets in {}'.format(
          len(rules), project.project_id)
      rules.append(
          _get_project_rule(name, 'whitelist', 'bucket', bucket_ids, bindings))

    return rules


def _get_global_whitelist_rule(resource_type, standard_members, bindings_list,
                               domain):
  """Get a global whitelist rule for the given resource.

  Args:
    resource_type (str): type of resource.
    standard_members (List[str]): generic list of members to create the
      whitelist for. The members can have wildcards.
    bindings_list(List[Dict[str, List[str]]]): list of bindings (map of
      role to wildcard members).
    domain (str): domain to create rules for.

  Returns:
    Dict: global whitelist rule.
  """
  formatted_members = [
      member.format(domain=domain)
      for member in standard_members
  ]

  # convert wildcarded members into a single regex
  escaped_members = [re.escape(member) for member in formatted_members]
  regex = re.compile('|'.join(escaped_members).replace('\\*', '.+'))

  # members not captured by the initial standard members
  unmatched_members = _get_unmatched_members_from_bindings(bindings_list, regex)

  rule = {
      'name': 'Global whitelist of allowed members for {} roles'.format(
          resource_type),
      'mode': 'whitelist',
      'resource': [{
          'type': resource_type,
          'applies_to': 'self',
          'resource_ids': ['*'],
      }],
      'inherit_from_parents': True,
      'bindings': [{
          'role': '*',
          'members': formatted_members + sorted(unmatched_members),
      }],
  }
  return rule


def _get_unmatched_members_from_bindings(bindings_list, regex):
  """Get all binding members that do not match the regex.

  Args:
    bindings_list(List[Dict[str, List[str]]]): List of binding dicts
      (role to members).
    regex (Regex): regex to match members to.

  Returns:
     Set[str]: members that did not match the given regex.
  """
  unmatched_members = set()
  for bindings in bindings_list:
    for members in bindings.values():
      unmatched_members.update(m for m in members if not regex.match(m))
  return unmatched_members


def _get_project_rule(name, mode, resource_type, resource_ids, binding_dict):
  """Helper function to build a rule dictionary."""
  # Use a stable ordering for roles to make comparing config diffs easier.
  rule_bindings = [{
      'role': role,
      'members': binding_dict[role] or [_NOBODY],
  } for role in sorted(binding_dict.keys())]
  return {
      'name': name,
      'mode': mode,
      'resource': [{
          'type': resource_type,
          'applies_to': 'self',
          'resource_ids': resource_ids,
      }],
      'inherit_from_parents': True,
      'bindings': rule_bindings,
  }
