"""Class holding configuration details of a monitored GCP project."""

from __future__ import absolute_import
from __future__ import division
from __future__ import print_function

import collections

# Name of the local audit logs dataset from the deployment template.
_LOCAL_AUDIT_LOGS_DATASET = 'audit_logs'
# Default Service Accounts with Editors role.
_EDITOR_SERVICE_ACCOUNTS = [
    # Compute Engine default service account.
    '{project_num}-compute@developer.gserviceaccount.com',
    # Google APIs Service Agent (e.g. Deployment manager).
    '{project_num}@cloudservices.gserviceaccount.com',
    # Google Container Registry Service Agent.
    'service-{project_num}@containerregistry.iam.gserviceaccount.com',
]
# Map from prefixes used by IAM to those used in BigQuery.
_IAM_TYPE_TO_BIGQUERY_TYPE = {
    'domain': 'domain',
    'group': 'group_email',
    'user': 'user_email',
    'serviceAccount': 'user_email',  # BigQuery treats serviceAccounts as users.
    'specialGroup': 'special_group',
}


def _group_name(group_email):
  """Gets an IAM member string for a group email."""
  return 'group:{}'.format(group_email)


def _service_account_name(service_acct_email):
  """Gets an IAM member string for a service account email."""
  return 'serviceAccount:{}'.format(service_acct_email)


def _get_bigquery_member(member):
  """Gets a BigQuery-style member given an IAM-style member string."""
  # IAM doesn't use a prefix for allAuthenticatedUsers, but BigQuery treats it
  # as a special_group (BigQuery doesn't support allUsers).
  if member == 'allAuthenticatedUsers':
    return {'special_group': member}
  member_type, member_name = member.split(':')
  return {_IAM_TYPE_TO_BIGQUERY_TYPE[member_type]: member_name}


def _create_bigquery_bindings(owners, writers, readers):
  """"Create Bigquery role-member bindings given lists of members."""
  return [
      {
          'role': 'OWNER',
          'members': [_get_bigquery_member(o) for o in owners],
      },
      {
          'role': 'WRITER',
          'members': [_get_bigquery_member(w) for w in writers],
      },
      {
          'role': 'READER',
          'members': [_get_bigquery_member(r) for r in readers],
      },
  ]


Bucket = collections.namedtuple('Bucket', ['id', 'location'])


class ProjectConfig(object):
  """Configuration for a single GCP project."""

  def __init__(self, overall, project, audit_logs_project):
    """Initialize.

    Args:
        overall (dict): The parsed dictionary of the overall/global
            configuration.
        project (dict): The parsed dictionary of a single project from the
            project config YAML file.
        audit_logs_project (dict): the audit logs project, or None if project is
            using local audit logs.
    """
    self._project_config = project
    self._uses_local_audit_logs = audit_logs_project is None

    self.project_id = self._project_config['project_id']
    self.enabled_apis = self._project_config.get('enabled_apis')

    self._forseti_gcp_reader = overall['forseti_gcp_reader']

    # List of default access groups, with 'group:' prefix.
    self._owners = [_group_name(self._project_config['owners_group'])]
    self._auditors = [_group_name(self._project_config['auditors_group'])]
    self._writers = [
        _group_name(w)
        for w in self._project_config.get('data_readwrite_groups', [])
    ]
    self._readers = [
        _group_name(r)
        for r in self._project_config.get('data_readonly_groups', [])
    ]

    # TODO: split audit logs into its own class to avoid duplication
    # of functions and vars
    self.audit_logs_bigquery_dataset = (
        self._project_config['audit_logs']['logs_bigquery_dataset'])
    if audit_logs_project:
      self.audit_logs_project_id = audit_logs_project['project_id']
      self._audit_logs_owners = [
          _group_name(audit_logs_project['owners_group'])]
    else:
      self.audit_logs_project_id = self.project_id
      self._audit_logs_owners = self._owners
      self.audit_logs_bigquery_dataset['name'] = _LOCAL_AUDIT_LOGS_DATASET

  def get_project_bindings(self):
    """Get expected IAM bindings at the project level.

    Returns:
      dict: a map from role name (str) to a list of members which should hold
          that role at the project level.
    """
    bindings = collections.defaultdict(list)
    bindings['roles/owner'] = self._owners
    # Editors will be default service accounts and explicitly provided editors
    # groups (if any).
    editors = [
        _group_name(g) for g in self._project_config.get('editors_groups', [])]
    project_num = self._project_config['generated_fields']['project_number']
    for service_account in _EDITOR_SERVICE_ACCOUNTS:
      editors.append(_service_account_name(service_account.format(
          project_num=project_num)))
    bindings['roles/editors'] = editors
    bindings['roles/iam.securityReviewer'] = (
        self._auditors + [_service_account_name(self._forseti_gcp_reader)])
    for additional in self._project_config.get('additional_project_permissions',
                                               []):
      for role in additional['roles']:
        bindings[role].extend(additional['members'])
    return bindings

  def get_buckets(self):
    """Get the GCS buckets in the project.

    Returns:
      List[Bucket]: The GCS buckets in the project.
    """
    buckets = [
        Bucket(
            id=self.project_id + bucket_dict['name_suffix'],
            location=bucket_dict['location'],
        )
        for bucket_dict in self._project_config.get('data_buckets', [])
    ]
    return buckets

  def get_audit_log_bucket(self):
    """Get the audit log GCS bucket.

    Returns:
      Bucket: The audit log GCS bucket or None if there is no bucket.
    """
    if 'logs_gcs_bucket' not in self._project_config.get('audit_logs', {}):
      return None

    bucket_dict = self._project_config['audit_logs']['logs_gcs_bucket']
    location = bucket_dict['location']
    bid = bucket_dict.get('name', '{}-logs'.format(self.project_id))
    return Bucket(id=bid, location=location)

  def get_bucket_bindings(self):
    """Returns a list of bucket names and their expected bindings."""
    bindings = []

    # Add logs bucket if using local logs.
    if (self._uses_local_audit_logs and
        'logs_gcs_bucket' in self._project_config['audit_logs']):
      bindings.append((['{}-logs'.format(self.project_id)], {
          'roles/storage.admin': self._owners,
          'roles/storage.objectAdmin': [],  # There should be no objectAdmins.
          'roles/storage.objectViewer': self._auditors,
          'roles/storage.objectCreator': [
              _group_name('cloud-storage-analytics@google.com')
          ],
      }))

    # Add data buckets, if any.
    if 'data_buckets' in self._project_config:
      # All data buckets by default have the same bindings.
      standard_data_buckets = []
      for data_bucket in self._project_config['data_buckets']:
        bucket_id = self.project_id + data_bucket['name_suffix']
        if 'additional_bucket_permissions' in data_bucket:
          extra = data_bucket['additional_bucket_permissions']
          # Bucket has extra, custom permissions.
          bindings.append(([bucket_id], {
              'roles/storage.admin': self._owners + extra.get('owners', []),
              'roles/storage.objectAdmin': (
                  self._writers + extra.get('readwrite', [])),
              'roles/storage.objectCreator': extra.get('writeonly', []),
              'roles/storage.objectViewer': (
                  self._readers + extra.get('readonly', [])),
          }))
        else:
          standard_data_buckets.append(bucket_id)
      if standard_data_buckets:
        bindings.append((standard_data_buckets, {
            'roles/storage.admin': self._owners,
            'roles/storage.objectAdmin': self._writers,
            'roles/storage.objectCreator': [],  # No objectCreators.
            'roles/storage.objectViewer': self._readers,
        }))

    return bindings

  def get_project_bigquery_bindings(self):
    """Returns a list of BigQuery datasets names and their expected bindings."""
    ids_and_bindings = []
    standard_datasets = []

    for dataset in self._project_config.get('bigquery_datasets', []):
      dataset_id = '{}:{}'.format(self.project_id, dataset['name'])
      if 'additional_dataset_permissions' in dataset:
        extra = dataset['additional_dataset_permissions']
        # Dataset has extra, custom permissions.
        ids_and_bindings.append(([dataset_id], _create_bigquery_bindings(
            self._owners + extra.get('owners', []),
            self._writers + extra.get('readwrite', []),
            self._readers + extra.get('readonly', []),
        )))
      else:
        standard_datasets.append(dataset_id)

    if standard_datasets:
      ids_and_bindings.insert(
          0,
          (standard_datasets,
           _create_bigquery_bindings(
               self._owners, self._writers, self._readers)))
    return ids_and_bindings

  def get_audit_logs_bigquery_bindings(self):
    """Returns a list of audit logs project bindings for BigQuery rules."""

    owners = self._audit_logs_owners
    writers = [
        _service_account_name(self._project_config['generated_fields']
                              ['log_sink_service_account'])
    ]
    readers = self._auditors

    return _create_bigquery_bindings(owners, writers, readers)

  def get_audit_log_sink_destination(self):
    """Returns the destination of the Audit Logs sink."""
    return 'bigquery.googleapis.com/projects/{}/datasets/{}'.format(
        self.audit_logs_project_id, self.audit_logs_bigquery_dataset['name'])

