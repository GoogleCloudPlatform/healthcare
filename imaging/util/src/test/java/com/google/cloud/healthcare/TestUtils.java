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

package com.google.cloud.healthcare;

import static org.junit.Assert.assertNotNull;

import com.google.common.io.ByteStreams;
import java.io.IOException;
import java.io.InputStream;
import org.dcm4che3.io.DicomInputStream;

/** Utility methods for tests. */
public class TestUtils {
  public static final String TEST_MR_FILE = "mr_1.0.0.0.dcm";
  public static final String TEST_IMG_FILE = "img_1.0.0.0.dcm";

  private TestUtils() {}

  public static byte[] readTestFile(String testFile) throws IOException {
    return ByteStreams.toByteArray(streamTestFile(testFile));
  }

  public static InputStream streamTestFile(String testFile) throws IOException {
    InputStream in = TestUtils.class.getClassLoader().getResourceAsStream(testFile);
    assertNotNull("test file missing: " + testFile, in);
    return in;
  }

  // Loads a DICOM file and strips file meta headers. This matches the behavior of how DICOM
  // binary instances are transmitted in DIMSE.
  // https://groups.google.com/forum/#!searchin/comp.protocols.dicom/file$20meta$20header%7Csort:date/comp.protocols.dicom/L1NL1VB6KTA/F9bH3GNf2EMJ
  public static InputStream streamDICOMStripHeaders(String testFile) throws IOException {
    DicomInputStream in = new DicomInputStream(TestUtils.streamTestFile(testFile));
    // Read and discard the meta-header in the stream (advances stream past meta header bytes).
    in.getFileMetaInformation();
    return in;
  }
}
