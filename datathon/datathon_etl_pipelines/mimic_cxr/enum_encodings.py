"""Enum encodings for the types and values in the MIMIC CXR dataset."""


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
