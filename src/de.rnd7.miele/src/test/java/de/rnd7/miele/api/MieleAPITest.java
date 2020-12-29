package de.rnd7.miele.api;

import junit.framework.TestCase;
import org.junit.Test;

import java.util.List;
import java.util.Objects;

public class MieleAPITest extends TestCase {


    @Test
    public void testLogin() throws Exception {
        final MieleAPI api = TestHelper.createAPI();
        assertNotNull("API token must be available", api.getToken());
    }

    @Test
    public void test_device_list() throws Exception {
        final MieleAPI api = TestHelper.createAPI();
        final List<MieleDevice> devices = api.fetchDevices();
        assertEquals(1, devices.size());
        final MieleDevice dishwasher = devices.iterator().next();
        assertEquals("G7560", dishwasher.getData()
                .getJSONObject("ident")
                .getJSONObject("deviceIdentLabel")
                .get("techType").toString());
    }
}