package de.rnd7.miele.api;

import junit.framework.TestCase;
import org.apache.commons.io.IOUtils;
import org.json.JSONObject;
import org.junit.Test;

import java.io.InputStream;
import java.nio.charset.StandardCharsets;

public class MieleDeviceTest extends TestCase {
    @Test
    public void testDeviceTurnedOff() throws Exception {
        try (final InputStream in = MieleDeviceTest.class.getResourceAsStream("device-off.json")) {
            final JSONObject device = new JSONObject(IOUtils.toString(in, StandardCharsets.UTF_8));
            final MieleDevice mieleDevice = new MieleDevice("device-id", device);

            JSONObject message = mieleDevice.toSmallMessage();
            assertEquals(0, message.get("remainingDurationMinutes"));
            assertEquals("OFF", message.get("phase").toString());
            assertEquals(0, message.get("phaseId"));
            assertEquals("OFF", message.get("state").toString());
        }
    }

    @Test
    public void testDeviceTurnedOn() throws Exception {
        try (final InputStream in = MieleDeviceTest.class.getResourceAsStream("device-on.json")) {
            final JSONObject device = new JSONObject(IOUtils.toString(in, StandardCharsets.UTF_8));
            final MieleDevice mieleDevice = new MieleDevice("device-id", device);

            JSONObject message = mieleDevice.toSmallMessage();
            assertEquals(82, message.getInt("remainingDurationMinutes"));
            assertEquals("FINISHED", message.get("phase").toString());
            assertEquals(1800, message.getInt("phaseId"));
            assertEquals("RUNNING", message.get("state").toString());
        }
    }
}