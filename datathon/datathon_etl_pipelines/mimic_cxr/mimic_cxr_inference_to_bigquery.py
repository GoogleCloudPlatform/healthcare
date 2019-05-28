"""Do inference with a MIMIC CXR keras model, and export results to BigQuery."""

import inspect
import os
from apache_beam.io.gcp.bigquery_tools import parse_table_schema_from_json
from datathon_etl_pipelines.generic_imaging.inference_to_bigquery import build_and_run_pipeline
from datathon_etl_pipelines.generic_imaging.inference_to_bigquery import ExampleWithImageBytesToInput
from datathon_etl_pipelines.generic_imaging.inference_to_bigquery import get_commandline_args
from datathon_etl_pipelines.generic_imaging.inference_to_bigquery import Predict
from datathon_etl_pipelines.mimic_cxr.enum_encodings import ID_NAMES
from datathon_etl_pipelines.mimic_cxr.enum_encodings import LABEL_NAMES
from datathon_etl_pipelines.mimic_cxr.prepare_mimic_cxr import ids_to_path


def example_to_ids(example):
  id_values = [
      example.features.feature[id_name].int64_list.value[0]
      for id_name in ID_NAMES
  ]
  result = dict(zip(ID_NAMES, id_values))
  result['path'] = ids_to_path(id_values)
  return result


def output_to_row(output_array):
  # x.item() converts numpy types to native python types.
  return dict(zip(LABEL_NAMES, (x.item() for x in output_array)))


def import_json_bq_schema():
  path = os.path.join(
      os.path.dirname(inspect.getfile(inspect.currentframe())),
      'mimic_cxr_bigquery_predictions_schema.json')

  with open(path) as fp:
    return parse_table_schema_from_json(fp.read())


def main():
  args, beam_options = get_commandline_args(description=__doc__)

  predict_dofn = Predict(
      keras_model_uri=args.keras_model,
      example_to_row=example_to_ids,
      example_to_input=ExampleWithImageBytesToInput(image_format='jpg'),
      output_to_row=output_to_row)

  build_and_run_pipeline(
      pipeline_options=beam_options,
      tfrecord_pattern=args.input_tfrecord_pattern,
      predict_dofn=predict_dofn,
      bq_table_schema=import_json_bq_schema(),
      output_bq_table=args.bigquery_table)


if __name__ == '__main__':
  main()
