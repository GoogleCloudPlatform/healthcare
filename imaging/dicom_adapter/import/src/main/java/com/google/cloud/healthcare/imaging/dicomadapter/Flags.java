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
  @Parameter(
    names = {"--dimse_aet"},
    description = "Title of DIMSE Application Entity."
  )
  String dimseAET = "";

  @Parameter(
    names = {"--dimse_port"},
    description = "Port the server is listening to for incoming DIMSE requests."
  )
  Integer dimsePort = 0;

  @Parameter(
    names = {"--dicomweb_addr"},
    description = "Address for DicomWeb service."
  )
  String dicomwebAddr = "";

  @Parameter(
    names = {"--dicomweb_stow_path"},
    description =
        "Path to send StowRS requests for DicomWeb peer. This is appended to the contents of --dicomweb_addr flag."
  )
  String dicomwebStowPath = "";

  @Parameter(
    names = {"--oauth_scopes"},
    description = "Comma seperated OAuth scopes used by adapter."
  )
  String oauthScopes = "";

  @Parameter(
    names = {"--verbose"},
    description = "Prints out debug messages."
  )
  boolean verbose = false;

  public Flags() {}
}
