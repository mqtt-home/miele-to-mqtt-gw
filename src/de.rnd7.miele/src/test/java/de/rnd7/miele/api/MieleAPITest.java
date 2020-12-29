package de.rnd7.miele.api;

import junit.framework.TestCase;
import org.junit.Test;

import java.util.List;

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
        assertFalse("Expected at least one device", devices.isEmpty());
        final MieleDevice dishwasher = devices.iterator().next();
        assertFalse("Expected device name to be non-empty", dishwasher.getData()
                .getJSONObject("ident")
                .getJSONObject("deviceIdentLabel")
                .get("techType").toString().isEmpty());
    }
}