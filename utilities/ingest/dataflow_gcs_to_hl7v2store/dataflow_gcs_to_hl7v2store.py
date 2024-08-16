#Importing Necessary Libraries
import google.auth
import google.auth.transport.requests
import requests
import time
import json
import logging
import base64
from requests.adapters import HTTPAdapter
from requests.packages.urllib3.util.retry import Retry
import apache_beam as beam
from apache_beam import pvalue
from apache_beam.options.pipeline_options import PipelineOptions
from apache_beam.options.pipeline_options import SetupOptions
from apache_beam import io
from datetime import datetime, timedelta
import argparse
import google

logging.basicConfig(level=logging.INFO, format='%(asctime)s :: %(levelname)s :: %(message)s')

class hcapi_hl7_store:

    def __init__(self, args:dict):
        self.hcapi_hl7_store = str(args['hcapi_hl7_store'])
        self.hcapi_project_id = str(args['hcapi_project_id'])
        self.hcapi_version = str(args['hcapi_version'])
        self.hcapi_location = str(args['hcapi_location'])
        self.hcapi_dataset = str(args['hcapi_dataset'])
        self.token = None

    # def google_api_headers(self):
    #     """ This function gets access tokens and authorizations\
    #         to access cloud healthcare API Fhir store"""
    #     creds, project = google.auth.default()
    #     auth_req = google.auth.transport.requests.Request()
    #     creds.refresh(auth_req)
    #     authToken = creds.token
    #     authHeaders = {
    #         "Authorization": f"Bearer {authToken}",
    #         "Prefer": "handling=strict"
    #         }
    #     return authHeaders

    def hcapi_dataset_url(self, version: str =None, project: str=None, location: str=None, dataset: str=None):
        """ This function creates base hcapi dataset url and returns it """
        """
        :param str version: HCAPI version from arguments passed
        :param str project: HCAPI project from arguments passed
        :param str location: HCAPI location from arguments passed
        :param str dataset: HCAPI dataset from arguments passed
        :return str hcapiDatasetURL: HCAPI dataset url build
        """
        base = "https://healthcare.googleapis.com"
        version = self.hcapi_version
        project = self.hcapi_project_id
        location = self.hcapi_location
        dataset = self.hcapi_dataset
        hcapiDatasetURL = f"{base}/{version}/projects/{project}/locations/{location}/datasets/{dataset}"
        return hcapiDatasetURL

    def createRequestSession(self):
        """Creating request session to try GET/POST requests using below force list"""
        reqSession = requests.Session()
        retries = Retry(total=3,
                        backoff_factor=2,
                        status_forcelist=[429, 500, 502, 503, 504, 400, 404, 401])

        reqSession.mount('http://', HTTPAdapter(max_retries=retries))
        return reqSession


    def hcapi_delete(self, url):
        """ Function to send delete request to HCAPI """
        response = requests.delete(url, headers=self.google_api_headers())
        if not response.ok:
            raise Exception(f'Error with HC API get:\n{response.text}')
        return response.json()

    def hcapi_hl7_url(self, version: str =None, project: str=None, location: str=None, dataset: str=None,store: str= None):
        """ This function creates hcapi hl7V2store url and returns the url """
        """
        :param str version: HCAPI version from arguments passed
        :param str project: HCAPI project from arguments passed
        :param str location: HCAPI location from arguments passed
        :param str dataset: HCAPI dataset from arguments passed
        :param str store: HCAPI FHIR store from arguments passed
        :return str hcapiHL7Url: HCAPI dataset url build
        """
        hcapiDatasetURL = self.hcapi_dataset_url(version=version, project=project,
        location=location, dataset=dataset)
        if store is None:
            raise Exception("No HL7V2 store specified")
        else:
            hl7_store = store
        hcapiHL7Url = f'{hcapiDatasetURL}/hl7V2Stores/{hl7_store}'
        return hcapiHL7Url
    
    def hcapi_get(self, url:str):
        """ This function to get HL7 Message from specified HL7V2 store"""
        """
        :param str url: HCAPI HL7 URL to fetch data
        :return dict responseJSON: response in JSON based on criteria
        :return str responseFlag: repsonse status pass or fail
        """
        try:
            responseFlag = "success"
            reqSession = self.createRequestSession()
            logging.info("creating headers with new token for HCAPI HL7V2 store GET request")
            creds, project = google.auth.default()
            auth_req = google.auth.transport.requests.Request()
            creds.refresh(auth_req)
            authToken = creds.token
            authHeaders = {
                "Authorization": f"Bearer {authToken}",
                "Prefer": "handling=strict"
            }
            logging.info("Fetching HL7V2 Message from HCAPI HL7V2 store")
            response = reqSession.get(url, headers=authHeaders,timeout=30)
            response.raise_for_status()
            responseJSON = response.json()
        except Exception as error:
            reqSession.close()
            responseFlag = "fail"
            logging.error("process to get HL7 Message failed")
            errorMessage = "Process to fetch HL7 Message from HCAPI HL7V2 store failed due to : {}".format(str(type(error).__name__)+" --> "+ str(error))
            responseJSON = dict()
            responseJSON['errorMessage'] = errorMessage
            return responseJSON,responseFlag
        else:
            reqSession.close()
            return responseJSON,responseFlag

    def hcapi_post(self, url:str, data:dict, source:str):
        """ This function to POST HL7 message resource from specified HL7V2 store """
        """
        :param str url: HCAPI HL7V2 store to fetch data
        :param str data: JSON payload string
        :param str source: specify source whether HL7 or FHIR
        :return dict responseJSON: response in JSON based on criteria
        :return str responseFlag: repsonse status pass or fail
        """
        try:
            responseFlag = "success"
            logging.info("creating headers with new token for HCAPI HL7V2 store POST request")
            creds, project = google.auth.default()
            auth_req = google.auth.transport.requests.Request()
            creds.refresh(auth_req)
            authToken = creds.token
            authHeaders = {
                "Authorization": f"Bearer {authToken}",
                "Prefer": "handling=strict"'message'
            }
            logging.info("Posting HL7 Message to HCAPI HL7V2 store")
            if source == "hl7":
                payload_data =data
            else:
                payload_data =json.loads(data)
            reqSession = self.createRequestSession()
            response = requests.post(url, headers=authHeaders, json=payload_data,timeout=30)
            logging.info("printing response:{}".format(str(response.text)))
            response.raise_for_status()
            responseJSON = response.json()
        except Exception as error:
            reqSession.close()
            logging.error("process to post HL7 Message failed")
            errorMessage = "Process to fetch HL7 Message from HCAPI HL7V2 store failed due to : {}".format(str(type(error).__name__)+" --> "+ str(error))
            responseJSON = dict()
            responseJSON['errorMessage'] = errorMessage
            responseFlag = "fail"
            return responseJSON,responseFlag
        else:
            reqSession.close()
            return responseJSON,responseFlag
    
    def post_hl7_message(self, message):
        """ Function to post messages to HL7V2 store """
        """
        :param str message: HL7 Message to post data
        :return dict responseJSON: response in JSON based on criteria
        :return str responseFlag: repsonse status pass or fail
        """
        messase =str(message)
        message = message.replace('\n', '\r')
        message = message.replace('\\r', '\r')
        message = message.replace('\r\r', '\r')
        encoded = base64.b64encode(message.encode())
        # logging.info(f"printing encoded message : {encoded}")
        payload = {
            "message": {
                "data": encoded.decode()
            }
        }
        url = self.hcapi_hl7_url(store=self.hcapi_hl7_store)+'/messages:ingest'
        logging.info("posting HL7v2Store using url : {}".format(url))
        postResponse,responseFlag = self.hcapi_post(url=url,data=payload,source = "hl7")
        return postResponse,responseFlag

    def get_hl7_message(self, message_id:str):
        """ Function to fetch messages from HL7V2 store"""
        """
        :param str url_path: FHIR resource URL to get FHIR resource from HCAPI FHIR store
        :return dict responseJSON: response in JSON based on criteria
        :return str responseFlag: repsonse status pass or fail
        """
        url_path=str(url_path)
        url = self.hcapi_hl7_url(store=self.hcapi_hl7_store)+f'/messages/{message_id}'
        responseJSON, responseFlag = self.hcapi_get(url)
        return responseJSON, responseFlag


class buildGCSFilePath(beam.DoFn):
    """ Class to get file name from variable and returns the filename """
    
    def process(self, element):
        # TO DO Remove before pushing to Git-Hub
        logging.info("printing raw element from pubsub")
        logging.info(f"{element}")
        logging.info("printing raw data element from pubsub")
        rawMessage = json.loads(element.data.decode('utf-8'))
        logging.info("printing raw pubsub message : {}".format(json.dumps(rawMessage)))
        logging.info("Printing attributes from pubsub message")
        messageAttributes = element.attributes
        logging.info("printing raw pubsub message : {}".format(json.dumps(messageAttributes)))
        gcspath = "gs://"+rawMessage["bucket"]+"/"+rawMessage["name"]
        logging.info("Processing the following file: {}".format(str(gcspath)))
        yield gcspath

class buildHL7Message(beam.DoFn):
    """ Class to read file, clean and seperate messgaes based on MSH segments """
    failTag = "failRecord"
    successTag = "successRecord"

    def process(self, file_path):
        try:
            logging.info("Starting to read file: {}".format(file_path))
            file = io.gcsio.GcsIO().open(filename=file_path, mode='r')
            read_file = file.read()
            new_file = str(read_file, encoding='utf-8').replace('\n', '\r')
            logging.info("Starting process read HL7 messages from file and building them")
            messages=[]
            for line in new_file.split('\r'):
                if line[:3] =='MSH':
                    messages.append(line)
                else:
                    messages[-1]+= line


            logging.info("Total number of messages parsed from the input file are {}".format(len(messages)))
            for message in messages:
                yield pvalue.TaggedOutput(self.successTag,message)
             
        except Exception as error :
            logging.error("Got the following error while building HL7 Message : {}".format('\n'+str(error)))
            errorMessage = "process to build HL7 Message failed while reading from file due to : {}".format(str(type(error).__name__)+" --> "+ str(error))
            errorPayload = dict()
            errorPayload['errorMessage'] = errorMessage
            errorPayload['rawMessage']= str(file_path)
            yield pvalue.TaggedOutput(self.failTag,errorPayload)

class PostToHL7V2Store(beam.DoFn):
    """ DoFn to POST HL7 message to HL7v2 store """
    failTag = "failRecord"
    successTag = "successRecord"

    def process(self, element):
        try:
            logging.info("starting to prepare and post message")          
            hl7v2_store_response,responseFlag = hcapi.post_hl7_message(element)
            message_id = hl7v2_store_response['message']['name'].split("/")[-1]
            logging.info("successfully posted message to Hl7v2 store with message id :- {}".format(message_id))
            yield pvalue.TaggedOutput(self.successTag,message_id)
        except Exception as error:
            logging.error("got the following error while posting message to HL7v2 store : {}".format(error))
            errorMessage = "process to post HL7 Message failed due to : {}".format(str(type(error).__name__)+" --> "+ str(error))
            errorPayload = dict()
            errorPayload['errorMessage'] = errorMessage
            errorPayload['rawMessage']= str(element)
            yield pvalue.TaggedOutput(self.failTag,errorPayload)

class writeToGCS(beam.DoFn):
    """ Class to write raw and errored messages to GCS """
    def process(self, element, bucket, foldername):
        """
         write raw / error messages to respective bucket
        :param str element: PCollection of message from consumeTransformMessages DoFn
        :param str bucket: bucket name
        :param str messageType: message type raw or error message
        :return: None message written to cloud storage
        :rtype: None
        """
        #TO DO below logging only for testing purpose delete when deploying to prod
        logging.info("printing raw element : {}".format(str(element)))
        
        currentTimestamp = datetime.now().strftime("%Y-%m-%d-%H%M%S")
        if foldername:
            folderName= str(foldername)+"-"+str(currentTimestamp)
        else:
            foldername = "undefined"
            folderName= str(foldername)+"-"+str(currentTimestamp)
        
        #converting incoming element to JSON
        # newElement = json.loads(json.dumps(element,separators=(',', ':')))
        newElement = str(element)
        

        if folderName and bucket:
            strElement = str(newElement).encode('utf-8')
            #Building GCS path based on current timestamp
            
            gcsPath = f"gs://{bucket}/{folderName}/{str(currentTimestamp[:10])}/{str(currentTimestamp.replace('-',''))}.txt"
            file = io.gcsio.GcsIO().open(filename=gcsPath, mode='w', mime_type='application/text')
            write_file = file.write(strElement)
            logging.info("successfully written message to cloud storage at :- {}".format(gcsPath))                
        else:
            logging.error("foldername and bucket not mentioned")

# TO DO: Include PubSub and test the pipeline end to end
def run(argsDict,beam_args,argv=None):
    beam_pipeline_options = PipelineOptions(beam_args,streaming=True,save_main_session=True)
    read_pubsub_subscription = str(argsDict['read_pubsub_subscription'])
    gcs_error_bucket = str(argsDict['gcs_error_bucket'])
    

    with beam.Pipeline(options=beam_pipeline_options) as pipeline:
        gcsFile = (
            pipeline
            | 'readFileNameFromPubSub' >> io.ReadFromPubSub(subscription=read_pubsub_subscription,with_attributes=True)
            | 'extractGCSFilePath' >> beam.ParDo(buildGCSFilePath())
        )
        processHL7Message=(
            gcsFile
            | 'parseHL7Message' >> beam.ParDo(buildHL7Message()).with_outputs()
            # | beam.Map(print)
        )
        postHL7Message =(
            processHL7Message.successRecord
            |'PostToHL7V2Store' >> beam.ParDo(PostToHL7V2Store()).with_outputs()
        )
        writeFailbuildRecords = (
            processHL7Message.failRecord
            | "writeFailBuildRecordsToGCS" >> beam.ParDo(writeToGCS(),bucket=gcs_error_bucket,foldername="build_error")
            )
        writeFailPostRecords = (
            processHL7Message.failRecord
            | "writeFailPostRecordsToGCS" >> beam.ParDo(writeToGCS(),bucket=gcs_error_bucket,foldername="post_error")
            )
        
if __name__ == "__main__":
    logging.getLogger().setLevel(logging.INFO)
    parser = argparse.ArgumentParser()
    parser.add_argument('--read_pubsub_subscription', required=True, help='Pub/Sub Subscription to get GCS filename')
    parser.add_argument('--gcs_error_bucket', required=True, help='GCS error bucket to store error messages')
    parser.add_argument('--hcapi_project_id', required=True, help='HCAPI project ID')
    parser.add_argument('--hcapi_dataset', required=True, help='HCAPI dataset')
    parser.add_argument('--hcapi_version', required=True, help='HCAPI Version')
    parser.add_argument('--hcapi_location', required=True, help='HCAPI location')
    parser.add_argument('--hcapi_hl7_store', required=True, help='HCAPI Hl7v2 store name')
    parser.add_argument('--hcapi_fhir_store', required=False, help='HCAPI FHIR Store name')
    args, beam_args = parser.parse_known_args()
    argsDict = vars(args)
    hcapi= hcapi_hl7_store(argsDict)
    run(argsDict=argsDict,beam_args=beam_args)
