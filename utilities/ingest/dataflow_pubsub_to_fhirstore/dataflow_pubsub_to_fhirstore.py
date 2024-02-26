#    Copyright 2023 Google LLC

#    Licensed under the Apache License, Version 2.0 (the "License");
#    you may not use this file except in compliance with the License.
#    You may obtain a copy of the License at

#        http://www.apache.org/licenses/LICENSE-2.0

#    Unless required by applicable law or agreed to in writing, software
#    distributed under the License is distributed on an "AS IS" BASIS,
#    WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
#    See the License for the specific language governing permissions and
#    limitations under the License.


##################################
# Author: Devansh Modi           #
##################################

#Main Dataflow MODULE :- fetching from PubSub, cleaning, filtering, transforming and posting to HCAPI FHIR store

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


#Below class is used to POST and GET FHIR resources from HealthCare API FHIR store (We will use Patient FHIR resource as an example)
class hcapi_fhir_store:
    def __init__(self, args:dict):
        self.hcapi_project_id = str(args['hcapi_project_id'])
        self.hcapi_version = str(args['hcapi_version'])
        self.hcapi_location = str(args['hcapi_location'])
        self.hcapi_dataset = str(args['hcapi_dataset'])
        self.hcapi_get_fhir_store = str(args['hcapi_get_fhir_store'])
        self.hcapi_post_fhir_store = str(args['hcapi_post_fhir_store'])
      

    def google_api_headers(self):
        """ This function gets access tokens and authorizations\
            to access cloud healthcare API """
        creds, project = google.auth.default()
        auth_req = google.auth.transport.requests.Request()
        creds.refresh(auth_req)
        authToken = creds.token
        authHeaders = {
            "Authorization": f"Bearer {authToken}",
            "Prefer": "handling=strict"
            }
        return authHeaders

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


    def hcapi_fhir_url(self, version: str =None, project: str=None, location: str=None, dataset: str=None,store: str= None):
        """ This function creates base hcapi FHIR url """
        """
        :param str version: HCAPI version from arguments passed
        :param str project: HCAPI project from arguments passed
        :param str location: HCAPI location from arguments passed
        :param str dataset: HCAPI dataset from arguments passed
        :param str store: HCAPI FHIR store from arguments passed
        :return str hcapiFHIRUrl: HCAPI dataset url build
        """
        hcapiDatasetURL = self.hcapi_dataset_url(version=version, project=project,
        location=location, dataset=dataset)
        if store is None:
            raise Exception("No FHIR store specified")
        else:
            store = store
        hcapiFHIRUrl = f'{hcapiDatasetURL}/fhirStores/{store}/fhir'
        return hcapiFHIRUrl
    
    def createRequestSession(self):
        """Creating request session to try GET/POST requests using below force list"""
        reqSession = requests.Session()
        retries = Retry(total=3,
                        backoff_factor=2,
                        status_forcelist=[429, 500, 502, 503, 504, 400, 404, 401])

        reqSession.mount('http://', HTTPAdapter(max_retries=retries))
        return reqSession

    def hcapi_get(self, url:str):
        """ This function to get FHIR resource from FHIR store specified """
        """
        :param str url: HCAPI FHIR URL to fetch data
        :return dict responseJSON: response in JSON based on criteria
        :return str responseFlag: repsonse status pass or fail
        """
        try:
            responseFlag = "success"
            reqSession = self.createRequestSession()
            logging.info("creating headers with new token for HCAPI FHIR store GET request")
            creds, project = google.auth.default()
            auth_req = google.auth.transport.requests.Request()
            creds.refresh(auth_req)
            authToken = creds.token
            authHeaders = {
                "Authorization": f"Bearer {authToken}",
                "Prefer": "handling=strict"
            }
            logging.info("Fetching FHIR resource from HCAPI FHIR store")
            response = reqSession.get(url, headers=authHeaders,timeout=30)
            response.raise_for_status()
            responseJSON = response.json()
        except Exception as error:
            reqSession.close()
            responseFlag = "fail"
            logging.error("process to get FHIR resource failed")
            errorMessage = "Process to fetch FHIR resource from HCAPI FHIR store failed due to : {}".format(str(type(error).__name__)+" --> "+ str(error))
            responseJSON = dict()
            responseJSON['errorMessage'] = errorMessage
            return responseJSON,responseFlag
        else:
            reqSession.close()
            return responseJSON,responseFlag

    def hcapi_post(self, url:str, data:str):
        """ This function to POST FHIR resource from FHIR store specified """
        """
        :param str url: HCAPI FHIR URL to fetch data
        :param str data: JSON payload string
        :return dict responseJSON: response in JSON based on criteria
        :return str responseFlag: repsonse status pass or fail
        """
        try:
            responseFlag = "success"
            logging.info("creating headers with new token for HCAPI FHIR store POST request")
            creds, project = google.auth.default()
            auth_req = google.auth.transport.requests.Request()
            creds.refresh(auth_req)
            authToken = creds.token
            authHeaders = {
                "Authorization": f"Bearer {authToken}",
                "Prefer": "handling=strict"
            }
            logging.info("Posting FHIR resource to HCAPI FHIR store")
            payload_data =json.loads(data)
            reqSession = self.createRequestSession()
            response = requests.post(url, headers=authHeaders, json=payload_data,timeout=30)
            response.raise_for_status()
            responseJSON = response.json()
        except Exception as error:
            reqSession.close()
            logging.error("process to POST FHIR resource failed")
            errorMessage = "Process to POST FHIR Resource to HCAPI FHIR store failed due to : {}".format(str(type(error).__name__)+" --> "+ str(error))
            responseJSON = dict()
            responseJSON['errorMessage'] = errorMessage
            responseFlag = "fail"
            return responseJSON,responseFlag
        else:
            reqSession.close()
            return responseJSON,responseFlag

    def post_fhir_message(self, resourceType:str,payload:str):
        """ Function to post messages to FHIR store """
        """
        :param str resourceType: FHIR Resource to post data
        :param str payload: FHIR Resource as string
        :return dict responseJSON: response in JSON based on criteria
        :return str responseFlag: repsonse status pass or fail
        """
        resourceType=str(resourceType)
        url = self.hcapi_fhir_url(store=self.hcapi_post_fhir_store)+f'/{resourceType}'
        postResponse,responseFlag = self.hcapi_post(url,payload)
        return postResponse,responseFlag

    #Below get_fhir_message function only kept for future reference not used in the script
    def get_fhir_message(self, url_path:str):
        """ Function to fetch FHIR resources from HCAPI FHIR store"""
        """
        :param str url_path: FHIR resource URL to get FHIR resource from HCAPI
        :return dict responseJSON: response in JSON based on criteria
        :return str responseFlag: repsonse status pass or fail
        """
        url_path=str(url_path)
        url = self.hcapi_fhir_url(store=self.hcapi_get_fhir_store)+f'/{url_path}'
        responseJSON, responseFlag = self.hcapi_get(url)
        return responseJSON, responseFlag


class transformations:
    """Generic class to build multiple FHIR resource as per business requirements.
        For example, we are building Patient FHIR resource """
    
    def buildIdentifier(message:dict, messageKey:str):
        """Function to build mulitple identifiers for a given FHIR resource"""
        """
        :param dict message: raw message
        :param str messageKey: identifier value example SSN, ID, etc.
        :return dict identifier: Identifier dictionary with system and value as per US Core
        """
        identifier= dict()
        identifier['system'] = str(messageKey)
        identifier['value'] = message[messageKey] if message[messageKey] else "N/A"
        return identifier

    @staticmethod
    def buildPatientResource(message:dict):
        """Function to create Patient FHIR resource from JSON payload receieved via ehr-portal-events Topic """
        """
        :param dict message: raw JSON payload dict
        :return dict patientResourceJSON: patient FHIR resource JSON
        :return str processFlag: process flag to indicate if the process failed or passed
        """
        try:
            patientResource = dict()
            patientResource['resourceType']='Patient'
            indentifier = list()
            indentifier.append(transformations.buildIdentifier(message,'SSN'))
            patientResource['identifier'] = indentifier
            patientResource['Id'] = message['Id'] if message['Id'] else "N/A"
            patientName = dict()
            patientName['use'] = 'official'
            patientName['family'] = message['LAST'] if message['LAST'] else "N/A"
            patientName['given'] =[message['FIRST'] if message['FIRST'] else "N/A"]
            patientResource['name'] = [patientName]
            patientResource['birthDate'] = message['BIRTHDATE'] if message['BIRTHDATE'] else "N/A"
            patientResource['gender'] = message['GENDER'] if message['GENDER'] else "N/A"
            address = dict()
            address['use'] = 'home'
            address['type'] = 'physical'
            address['line'] = [message['ADDRESS'] if message['ADDRESS'] else "N/A"]
            address['district'] = message['COUNTY'] if message['COUNTY'] else "N/A"
            address['city'] = message['CITY'] if message['CITY'] else "N/A"
            address['state'] = message['STATE'] if message['STATE'] else "N/A"
            address['country'] = message['COUNTRY'] if message['COUNTRY'] else "N/A"
            patientResource['address'] = [address]
            telecom = dict()
            telecom['use'] = 'home'
            telecom['system'] = 'email'
            telecom['value'] = message['EMAIL'] if message['EMAIL'] else "N/A"
            patientResource['telecom'] = [telecom]
            patientResourceJSON = json.loads(json.dumps(patientResource,separators=(',', ':')))
            logging.info("process to build patient FHIR resource completed")
            logging.info(f"printing transform Patient FHIR resource {json.dumps(patientResourceJSON)}")
            processFlag = "success"
        except Exception as error :
            logging.warning("process to build Patient FHIR Resource failed due to : {}".format(error, exc_info=True))
            errorMessage = "Process to build Patient FHIR Resource failed due to : {}".format(str(type(error).__name__)+" --> "+ str(error))
            rawMessage = dict()
            rawMessage['rawMessage'] = message
            rawMessage['errorMessage'] = errorMessage
            processFlag = "fail"
            return rawMessage,processFlag
        else:
            return patientResourceJSON,processFlag
            

#Below class is used to consume messages from Pub/Sub and transform it
class consumeTransformMessages(beam.DoFn):
    """ Beam DoFn to transform raw data and create FHIR resource as per Transformation"""
    #beam output tag to mark transformed element as pass record or fail record
    failTag = "failRecord"
    successTag = "successRecord"
    
    def process(self,element):
        """
        Note:- You can reuse this class to add/remove attributes based on payload receieved via pub/sub
        process to consume messages from pub/sub subscription and construct raw message JSON with attributes.
        There are multiple logging statements in place for debugging purpose one can remove as per requirements
        :return: transformed raw JSON
        :rtype: dict
        """
        try:
            logging.info("printing raw element from pubsub")
            logging.info(f"{element}")
            logging.info("printing raw data element from pubsub")
            rawMessage = json.loads(element.data.decode('utf-8'))
            logging.info(json.dumps(rawMessage))
            logging.info("Printing attributes from pubsub message")
            messageAttributes = element.attributes
            logging.info(json.dumps(messageAttributes))
            rawMessage['topic'] = messageAttributes['topic']
            rawMessage['timestamp'] = messageAttributes['timestamp']
            yield pvalue.TaggedOutput(self.successTag,rawMessage)
        except Exception as error :
            logging.error("process to transform pub/sub record record failed due to : {}".format(error, exc_info=True))
            errorMessage = "process to transform pub/sub record record failed due to : {}".format(str(type(error).__name__)+" --> "+ str(error))
            transformedPayload = dict()
            transformedPayload['errorMessage'] = errorMessage
            transformedPayload['rawMessage']= str(element)
            yield pvalue.TaggedOutput(self.failTag,transformedPayload)
    
class writeToGCS(beam.DoFn):
    """ Class to write raw and errored messages to GCS """
    def process(self, element, bucket, messageType, topic=None):
        """
         write raw / error messages to respective bucket
        :param str element: PCollection of message from consumeTransformMessages DoFn
        :param str bucket: bucket name
        :param str messageType: message type raw or error message
        :return: None message written to cloud storage
        :rtype: None
        """
        #TO DO below logging only for testing purpose delete when deploying to prod
        logging.info("printing raw element : {}".format(str(json.dumps(element))))
        
        if messageType :
            folderName= str(messageType)+"_messages"
        else:
            folderName= "unidentified"
        
        #converting incoming element to JSON
        newElement = json.loads(json.dumps(element,separators=(',', ':')))
        if topic:
            topicName = str(topic)
        elif 'topic' in newElement.keys():
            topicName = str(newElement['topic'])
        else:
            logging.info("topic name not mentioned or not present in topic")
        
        if topicName:
            if folderName and bucket:
                strJsonElement = str(newElement).encode('utf-8')
                #Building GCS path based on current timestamp
                currentTimestamp = datetime.now().strftime("%Y-%m-%d-%H%M%S")
                gcsPath = f"gs://{bucket}/{folderName}/{topicName}/{str(currentTimestamp[:10])}/{str(currentTimestamp.replace('-',''))}.json"
                file = io.gcsio.GcsIO().open(filename=gcsPath, mode='w', mime_type='application/json')
                write_file = file.write(strJsonElement)
                logging.info("successfully written message to cloud storage at :- {}".format(gcsPath))                
            else:
                logging.error("foldername and bucket not mentioned")
        else:
            logging.error("topic name not mentioned or not present in topic")
    
            
class postToHCAPI(beam.DoFn):
    """ Class to POST FHIR message to HCAPI """
    #beam output tag to mark transformed element as pass and fail
    failTag = "failRecord"

    def process(self, element,resourceType):
        logging.info("starting to prepare and post messages")
        dataJson = json.dumps(element,separators=(',', ':'))
        postResponse,postResponseFlag  = hcapi.post_fhir_message(resourceType=str(resourceType),payload=dataJson)
        if postResponseFlag == "success":
            logging.info("successfully written FHIR Resource to given FHIR store")
        elif postResponseFlag == "fail":
            logging.error("failed to write (POST) transformed FHIR Resource to given FHIR store")
            rawMessage = json.loads(json.dumps(element,separators=(',', ':')))
            rawMessage['postHCAPIErrorMessage'] = postResponse
            yield pvalue.TaggedOutput(self.failTag,rawMessage)


class buildFHIRResources(beam.DoFn):
    """ Beam DoFn to transform raw data and create careplan FHIR resource"""
    #beam output tag to mark transformed element as pass and fail
    failTag = "failRecord"
    successTag = "successRecord"
    def process(self, element,resourceType):
        try:
            logging.info("starting to build FHIR resource")
            if resourceType.lower() == "patient":
                patientResource,processFlag = transform.buildPatientResource(element)
                if processFlag == "success":
                    yield pvalue.TaggedOutput(self.successTag,patientResource)
                else:
                    logging.warning(f"failed to create  FHIR resource")
                    yield pvalue.TaggedOutput(self.failTag,patientResource)
            else:
                loggging.warning(f"please add the transformations for the given {resourceType} FHIR resource")
                yield pvalue.TaggedOutput(self.failTag,element)
        except Exception as error :
            logging.warning("process to build FHIR resource for {resourceType} FHIR resource failed due to : {}".format(error, exc_info=True))
            error = "process to build FHIR resource for {resourceType} FHIR resource failed due to :{}".format(str(type(error).__name__)+" --> "+str(error))
            errorMessage = dict()
            errorMessage['errorMessage'] = error
            errorMessage['rawMessage']= str(element)
            yield pvalue.TaggedOutput(self.failTag,errorMessage)
    
    
            
            
def run(argsDict,beam_args,argv=None):
    beam_pipeline_options = PipelineOptions(beam_args,streaming=True,save_main_session=True)
    read_pubsub_subscription = str(argsDict['read_pubsub_subscription'])
    gcs_archive_bucket = str(argsDict['gcs_archive_bucket'])
    gcs_error_bucket = str(argsDict['gcs_error_bucket'])
    
    with beam.Pipeline(options=beam_pipeline_options) as pipeline:
        consumeMessages = (
            pipeline
            | "readMessageFromPubSub" >> io.ReadFromPubSub(subscription=read_pubsub_subscription,with_attributes=True)
            | 'transformRawMessage' >> beam.ParDo(consumeTransformMessages()).with_outputs()
            )
        WriteSuccessRecordsToGCS = (
            consumeMessages.successRecord
            | "archiveRawMessageToGCS" >> beam.ParDo(writeToGCS(),bucket=gcs_archive_bucket,messageType="raw_success")
        )
        processSuccessRecords = (
            consumeMessages.successRecord
            | "buildFHIRResource" >> beam.ParDo(buildFHIRResources(),resourceType="Patient").with_outputs()
            )
        postResources = (
            processSuccessRecords.successRecord
            | "POST Success message to FHIR store" >> beam.ParDo(postToHCAPI(),resourceType="Patient").with_outputs()
            )
        writeFailRawRecords = (
            consumeMessages.failRecord
            | "writeFailRawRecordsToGCS" >> beam.ParDo(writeToGCS(),topic="fail_transform_raw_messages",bucket=gcs_error_bucket,messageType="error")
            )
        writeFailTransformRecords = (
            processSuccessRecords.failRecord
            | "writeFailTransformRecordsToGCS" >> beam.ParDo(writeToGCS(),topic="fail_transform_raw_messages",bucket=gcs_error_bucket,messageType="error")
            )
        writeFailPostRecords = (
            postResources.failRecord
            | "writeFailPostRecordsToGCS" >> beam.ParDo(writeToGCS(),topic="fail_hcapi_post_messages",bucket=gcs_error_bucket,messageType="error")
            )
        
if __name__ == "__main__":
    logging.basicConfig(level=logging.INFO, format='%(asctime)s :: %(levelname)-8s :: [%(filename)s:%(lineno)d] :: %(message)s')
    parser = argparse.ArgumentParser()
    parser.add_argument('--read_pubsub_subscription', required=True, help='Pub/Sub Subscription to consume messages')
    parser.add_argument('--hcapi_project_id', required=True, help='HCAPI project ID')
    parser.add_argument('--hcapi_dataset', required=True, help='HCAPI dataset')
    parser.add_argument('--hcapi_version', required=True, help='HCAPI Version')
    parser.add_argument('--hcapi_location', required=True, help='HCAPI location')
    parser.add_argument('--hcapi_get_fhir_store', required=False, help='HCAPI FHIR Store to GET FHIR resources')
    parser.add_argument('--hcapi_post_fhir_store', required=True, help='HCAPI FHIR Store to POST FHIR resources')
    parser.add_argument('--gcs_archive_bucket', required=True, help='GCS archive bucket to store raw messages')
    parser.add_argument('--gcs_error_bucket', required=True, help='GCS error bucket to store error messages')
    args, beam_args = parser.parse_known_args()
    argsDict = vars(args)
    hcapi= hcapi_fhir_store(argsDict)
    transform = transformations()
    run(argsDict=argsDict,beam_args=beam_args)
