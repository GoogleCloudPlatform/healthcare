#    Copyright 2024 Google LLC

#    Licensed under the Apache License, Version 2.0 (the "License");
#    you may not use this file except in compliance with the License.
#    You may obtain a copy of the License at

#        http://www.apache.org/licenses/LICENSE-2.0

#    Unless required by applicable law or agreed to in writing, software
#    distributed under the License is distributed on an "AS IS" BASIS,
#    WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
#    See the License for the specific language governing permissions and
#    limitations under the License.


import argparse
import logging
import json
import random
from datetime import datetime
import time
import pytz

import apache_beam as beam
from apache_beam.io import ReadFromPubSub
from apache_beam.options.pipeline_options import PipelineOptions
from apache_beam.transforms.window import FixedWindows
from google.cloud import bigquery
from apache_beam.io import fileio


class QueryRecordsFromBigQueryFn(beam.DoFn):
    """Query records from BigQuery using resource id and last_updated timestamp""""
    
    def __init__(self, table):
        self.table_name=table

    def setup(self):
        self.client=bigquery.Client()
    
    def construct_query(self, id_date_dict):
        where_condition_list = [
            f"(id='{k}' and FORMAT_DATETIME('%Y-%m-%d %H:%M:%S', meta.lastUpdated)="
            f"replace('{v}', '.000000', ''))"
            for k, v in id_date_dict.items()  
            ]
        filter_clause = " or ".join(where_condition_list)
        # Using careplan table in the exmaple. User can change this to anothe table or view
        query=f"""select id,status,intent from
                  {self.table_name} a
                  where {filter_clause}"""
        logging.info("Constructed Query: " + query)
        return query


    def process(self, id_date_dict):
        logging.info(id_date_dict)
        logging.info("constructing query")
        query=self.construct_query(id_date_dict)
        logging.info(query)
        time.sleep(3)
        query_job = self.client.query(query)
        for row in query_job:
            yield row[0] + "," + row[1] + "," + row[2]

class ProcessPubSubMessage(beam.DoFn):
    """Parse each PubSub message and extract the payload."""
    def process(self, grouped_msgs):
        id_date_dict = {}
        input_format = "%a, %d %b %Y %H:%M:%S %Z"
        output_format = "%Y-%m-%d %H:%M:%S.%f%Z"
        logging.info(grouped_msgs)
        for msg in grouped_msgs[1]:
            payload = msg.data.decode('utf-8')
            if ((msg.attributes['action'] == 'UpdateResource' or msg.attributes['action'] == 'CreateResource')  and msg.attributes['resourceType'] == 'CarePlan'):
                if msg.attributes['payloadType'] == 'NameOnly':
                    resource_id = payload.split('/')[-1]
                elif msg.attributes['payloadType'] == 'FullResource':
                    resource_id = json.loads(payload)['id']
                datetime_obj = datetime.strptime(msg.attributes['lastUpdatedTime'], input_format)
                last_updated_datetime = datetime_obj.strftime(output_format)
                id_date_dict[resource_id] = last_updated_datetime
        logging.info("Extracted Ids and the last updated time : " + str(id_date_dict))

        if id_date_dict:
            logging.info("Extracted Ids and the last updated time successfully : " + str(id_date_dict.keys()))
            yield id_date_dict
        else:
            logging.info("No ids are extracted since messages didn't match the pub/sub filter criteria")

def run(argv=None, save_main_session=True):
    """Main entry point; defines and runs the pipeline."""
    est_timezone = pytz.timezone('US/Eastern')
    parser = argparse.ArgumentParser()
    parser.add_argument(
        '--input_pubsub_subscription',
        dest='input_pubsub_subscription',
        required=True,
        help='Input subscription to consume data from.')
    parser.add_argument(
        '--bq_table',
        dest='bq_table',
        required=True,
        help='Input BQ table')
    parser.add_argument(
        '--gcs_output',
        dest='gcs_output',
        required=True,
        help='Output GCS path')
    known_args, pipeline_args = parser.parse_known_args(argv)

    beam_pipeline_options = PipelineOptions(pipeline_args,streaming=True,save_main_session=True)

    with beam.Pipeline(options=beam_pipeline_options) as p:
        processPubsubMessage = (
            p 
            | 'Read from Pub/Sub Subscription' >> ReadFromPubSub(subscription=known_args.input_pubsub_subscription, with_attributes=True)
            | 'Apply windowing' >> beam.WindowInto(FixedWindows(60)) # 60 seconds window Please change as per requirement
            | 'Add Key to messages' >> beam.WithKeys(lambda _: random.randint(0, 5)) # Using 0-5 rnadomy keys to group messages. Please change as per requirement
            | 'Group By Key' >> beam.GroupByKey()
            | 'Process PubSub messages' >> beam.ParDo(ProcessPubSubMessage())
            | 'Query Records From Bigquery' >> beam.ParDo(QueryRecordsFromBigQueryFn(known_args.bq_table))
            | 'Write to Cloud Storage' >> fileio.WriteToFiles(known_args.gcs_output + "/" +datetime.now(est_timezone).date().isoformat(), destination='csv')
            
        )



if __name__ == '__main__':
    run()

