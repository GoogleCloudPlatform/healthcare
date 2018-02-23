// Copyright 2018 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package com.google.cloud.healthcare.imaging.dicomadapter;

import com.beust.jcommander.Parameter;
import com.beust.jcommander.Parameters;

@Parameters(separators = "= ")
public class Flags {
  /** Flags for exporting via DIMSE C-Store. */
  @Parameter(
    names = {"--peer_dimse_aet"},
    description = "Application Entity Title of DIMSE peer."
  )
  public static String peerDimseAET = "";

  @Parameter(
    names = {"--peer_dimse_ip"},
    description = "IP of DIMSE peer."
  )
  public static String peerDimseIP = "";

  @Parameter(
    names = {"--peer_dimse_port"},
    description = "Port of DIMSE peer."
  )
  public static Integer peerDimsePort = 0;

  /** Flags for exporting via DicomWeb STOW-RS. */
  @Parameter(
    names = {"--peer_dicomweb_addr"},
    description = "Address of peer DicomWeb API serving STOW-RS."
  )
  public static String peerDicomwebAddr = "";

  @Parameter(
    names = {"--peer_dicomweb_stow_path"},
    description =
        "Path to send StowRS requests for DicomWeb peer. This is appended to the contents of --peer_dicomweb_addr flag."
  )
  public static String peerDicomwebStowPath = "";

  @Parameter(
    names = {"--use_gcp_application_default_credentials"},
    description =
        "If true, uses GCP Application Credentials to when sending HTTP requests to peer. This is useful if the peer DICOMWeb endpoint is running in GCP."
  )
  public static boolean useGcpApplicationDefaultCredentials = false;

  /** General flags applicable to both methods of export. */
  @Parameter(
    names = {"--dicomweb_addr"},
    description = "Address for DicomWeb service that sources the DICOM."
  )
  public static String dicomwebAddr = "";

  @Parameter(
    names = {"--project_id"},
    description = "Project ID for Cloud Pub/Sub messages."
  )
  public static String projectId = "";

  @Parameter(
    names = {"--subscription_id"},
    description = "Cloud Pub/Sub subscription ID of incoming messages."
  )
  public static String subscriptionId = "";

  @Parameter(
    names = {"--oauth_scopes"},
    description = "Comma seperated OAuth scopes used by adapter."
  )
  public static String oauthScopes = "";

  public Flags() {}
}
