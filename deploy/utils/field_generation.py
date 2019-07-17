"""field_generation provides utilities to manage generated_fields."""

from __future__ import absolute_import
from __future__ import division
from __future__ import print_function

from deploy.utils import utils

# The tag name of generated_fields in the new format.
# Using different variable with the old one so that we can easily change a
# different tag name while just changing the value of this variable.
GENERATED_FIELDS_NAME = 'generated_fields'
# The tag name of generated_fields in the old format.
_GENERATED_FIELDS_OLD_NAME = 'generated_fields'
_PROJECTS_TAG = 'projects'
_FORSETI_TAG = 'forseti'


def is_generated_fields_exist(project_id, input_config):
  """Check if generated_fields contains a project.

  Args:
    project_id (str): id of the project.
    input_config (CommentedMap): The content of the whole yaml.

  Returns:
    bool: True if exist, otherwise False.
  """
  return project_id in input_config.get(GENERATED_FIELDS_NAME,
                                        {}).get(_PROJECTS_TAG, {})


def get_generated_fields_copy(project_id, input_config):
  """Get a project's generated_field copy.

  Args:
    project_id (str): id of the project.
    input_config (CommentedMap): The content of the whole yaml.

  Returns:
    CommentedMap: Generated_field copy of the project.
  """
  if is_generated_fields_exist(project_id, input_config):
    return get_generated_fields_ref(project_id, input_config, False).copy()
  else:
    return {}


def get_generated_fields_ref(project_id, input_config, auto_create=True):
  """Get a project's generated_field reference that can be modified outside.

  If the project does not have a generated_field, then create one.

  Args:
    project_id (str): id of the project.
    input_config (CommentedMap): The content of the whole yaml.
    auto_create (bool): True if create when not exist.

  Returns:
    CommentedMap: Generated_field reference of the project.
  """
  if not auto_create and not is_generated_fields_exist(project_id,
                                                       input_config):
    raise utils.InvalidConfigError('%s generated_field does not exist.' %
                                   (project_id))
  if GENERATED_FIELDS_NAME not in input_config:
    input_config[GENERATED_FIELDS_NAME] = {}
  if _PROJECTS_TAG not in input_config[GENERATED_FIELDS_NAME]:
    input_config[GENERATED_FIELDS_NAME][_PROJECTS_TAG] = {}
  if project_id not in input_config[GENERATED_FIELDS_NAME][_PROJECTS_TAG]:
    input_config[GENERATED_FIELDS_NAME][_PROJECTS_TAG][project_id] = {}
  return input_config[GENERATED_FIELDS_NAME][_PROJECTS_TAG][project_id]


def is_deployed(project_id, input_config):
  """Determine whether the project has been deployed."""
  generated_fields = get_generated_fields_copy(project_id, input_config)
  if not generated_fields:
    return False
  else:
    return 'failed_step' not in generated_fields


def get_forseti_service_generated_fields(input_config):
  """Get generated_fields containing forseti service info."""
  return input_config.get(GENERATED_FIELDS_NAME, {}).get(_FORSETI_TAG, {})


def set_forseti_service_generated_fields(forseti_generated_fields,
                                         input_config):
  """Set generated_fields containing forseti service info."""
  if GENERATED_FIELDS_NAME not in input_config:
    input_config[GENERATED_FIELDS_NAME] = {}
  input_config[GENERATED_FIELDS_NAME][_FORSETI_TAG] = forseti_generated_fields


def update_generated_fields(input_yaml_path, new_config):
  """Get and update the generated_fields block of the input yaml."""
  cfg_content = utils.read_yaml_file(input_yaml_path)
  if GENERATED_FIELDS_NAME not in new_config:
    cfg_content.pop(GENERATED_FIELDS_NAME, {})
  else:
    cfg_content[GENERATED_FIELDS_NAME] = new_config[GENERATED_FIELDS_NAME]
  return cfg_content


def rewrite_generated_fields_back(project_yaml, new_config):
  """Write config file to output_yaml_path with new generated_fields."""
  cfg_content = update_generated_fields(project_yaml, new_config)
  utils.write_yaml_file(cfg_content, project_yaml)
