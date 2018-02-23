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

import java.io.ByteArrayInputStream;
import java.io.ByteArrayOutputStream;
import java.io.IOException;
import java.io.InputStream;
import java.io.SequenceInputStream;
import org.dcm4che3.data.Attributes;
import org.dcm4che3.data.UID;
import org.dcm4che3.io.DicomOutputStream;
import org.dcm4che3.net.PDVInputStream;

/** Provides utilities for handling DICOM streams. */
public class DicomStreamUtil {

  // Adds the DICOM meta header to input stream.
  public static InputStream dicomStreamWithFileMetaHeader(
      String sopInstanceUID,
      String sopClassUID,
      String transferSyntax,
      PDVInputStream inDicomStream)
      throws IOException {
    // File meta header (group 0002 tags), always in Explicit VR Little Endian.
    // http://dicom.nema.org/dicom/2013/output/chtml/part10/chapter_7.html
    ByteArrayOutputStream outBuffer = new ByteArrayOutputStream();
    DicomOutputStream fmiStream = new DicomOutputStream(outBuffer, UID.ExplicitVRLittleEndian);
    Attributes fmi =
        Attributes.createFileMetaInformation(sopInstanceUID, sopClassUID, transferSyntax);
    fmiStream.writeFileMetaInformation(fmi);

    // Add the file meta header + DICOM dataset (other groups) as a sequence of input streams.
    return new SequenceInputStream(
        new ByteArrayInputStream(outBuffer.toByteArray()), inDicomStream);
  }

  private DicomStreamUtil() {}
}
