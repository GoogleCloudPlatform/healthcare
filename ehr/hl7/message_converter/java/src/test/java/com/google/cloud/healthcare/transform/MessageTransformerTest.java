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

package com.google.cloud.healthcare.transform;

import static org.junit.Assert.assertEquals;
import static org.powermock.api.mockito.PowerMockito.mockStatic;
import static org.powermock.api.mockito.PowerMockito.when;

import com.google.common.io.ByteStreams;
import com.google.common.io.CharStreams;
import java.io.IOException;
import java.io.InputStreamReader;
import java.util.UUID;
import org.junit.Before;
import org.junit.Test;
import org.junit.runner.RunWith;
import org.powermock.core.classloader.annotations.PrepareForTest;
import org.powermock.modules.junit4.PowerMockRunner;

@RunWith(PowerMockRunner.class)
@PrepareForTest(MessageTransformer.class)
public class MessageTransformerTest extends TestBase {
  private static final UUID FAKE_UUID = UUID.fromString("38dd0ea4-a734-423c-8ff5-6a736d578dc4");

  private MessageTransformer transformer = new MessageTransformer();

  @Before
  public void setUp() {
    mockStatic(UUID.class);
    when(UUID.randomUUID()).thenReturn(FAKE_UUID);
  }

  @Test
  public void transformAdtA01() throws IOException {
    byte[] msg = ByteStreams.toByteArray(loadTestFile("adt_a01.hl7"));
    String expected = CharStreams.toString(new InputStreamReader(loadTestFile("adt_a01.json")));
    assertEquals(expected, transformer.transform(msg));
  }

  @Test
  public void transformOruR01() throws IOException {
    byte[] msg = ByteStreams.toByteArray(loadTestFile("oru_r01.hl7"));
    String expected = CharStreams.toString(new InputStreamReader(loadTestFile("oru_r01.json")));
    assertEquals(expected, transformer.transform(msg));
  }

  @Test(expected = UnsupportedOperationException.class)
  public void transformUnsupportedMsg() throws IOException {
    byte[] msg = ByteStreams.toByteArray(loadTestFile("siu_s12.hl7"));
    transformer.transform(msg);
  }
}
