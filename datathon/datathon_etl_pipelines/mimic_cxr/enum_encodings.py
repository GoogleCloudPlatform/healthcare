"""Enum encodings for the types and values in the MIMIC CXR dataset."""
import re
from typing import Dict, Union


class SerializableEnum(object):
  """A replacement for enum.Enum that can be serialized by Apache Beam."""

  def __init__(self, names):
    self.values = names
    self.index_dict = dict(zip(names, range(len(names))))

  def __getitem__(self, name):
    # type: (str) -> int
    return self.index_dict[name]

  def __call__(self, i):
    # type: (int) -> str
    return self.values[i]

  def __iter__(self):
    for v in self.values:
      yield v

  def __len__(self):
    return len(self.values)


LABEL_NAMES = SerializableEnum([
    'no_finding',
    'enlarged_cardiomediastinum',
    'cardiomegaly',
    'airspace_opacity',
    'lung_lesion',
    'edema',
    'consolidation',
    'pneumonia',
    'atelectasis',
    'pneumothorax',
    'pleural_effusion',
    'pleural_other',
    'fracture',
    'support_devices',
])

VIEW_VALUES = SerializableEnum([
    'frontal',
    'lateral',
    'other',
])

LABEL_VALUES = SerializableEnum([
    'not_mentioned',
    'negative',
    'uncertain',
    'positive',
])

DATASET_VALUES = SerializableEnum([
    'train',
    'valid',
])

ID_NAMES = SerializableEnum(['patient', 'study', 'image', 'view', 'dataset'])


def path_to_ids(path):
  """Parse the path to an image in the dataset.

  Args:
    path (str): A relative path to a JPG image.

  Returns:
    Tuple[int, int, int, int, int]: the patient_id, study_id, image_id, view,
    and dataset for this image.
  """
  match = re.match(
      r'(train|valid)/p([0-9]+)/s([0-9]+)/view([0-9]+)'
      r'_(frontal|lateral|other)\.jpg', path)
  if match is not None:
    d, pid, sid, iid, v = match.groups()
    patient_id = int(pid)
    study_id = int(sid)
    image_id = int(iid)
    view = VIEW_VALUES[v]
    dataset = DATASET_VALUES[d]
    return patient_id, study_id, image_id, view, dataset
  raise ValueError('unrecognized path: {}'.format(path))


def ids_to_path(ids):
  id_dict = dict(zip(ID_NAMES, ids))  # type: Dict[str, Union[int, str]]
  id_dict['view'] = VIEW_VALUES(id_dict['view'])
  id_dict['dataset'] = DATASET_VALUES(id_dict['dataset'])
  return '{dataset}/p{patient}/s{study:02d}/view{image}_{view}.jpg'.format(
      **id_dict)
