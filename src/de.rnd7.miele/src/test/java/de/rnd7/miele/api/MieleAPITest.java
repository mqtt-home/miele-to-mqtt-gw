package de.rnd7.miele.api;

import junit.framework.TestCase;
import org.junit.Test;

import java.util.List;
import java.util.Objects;

public class MieleAPITest extends TestCase {

    private String forceEnv(String propName) {
        // macOS note: sudo vi /etc/launchd.conf
        final String value = Objects.requireNonNull(System.getenv(propName),
                String.format("ENV property %s is required to run this test case.", propName));

        if (value.trim().isEmpty()) {
            throw new IllegalArgumentException(String.format("ENV property %s must not be empty to run this test case.", propName));
        }

        return value;
    }

    private MieleAPI createAPI() {
        return new MieleAPI(
            forceEnv("MIELE_CLIENT_ID"),
            forceEnv("MIELE_CLIENT_SECRET"),
            forceEnv("MIELE_USERNAME"),
            forceEnv("MIELE_PASSWORD")
        );
    }

    @Test
    public void testLogin() throws Exception {
        final MieleAPI api = createAPI();
        assertNotNull("API token must be available", api.getToken());
    }

    @Test
    public void test_device_list() throws Exception {
        final MieleAPI api = createAPI();
        final List<MieleDevice> devices = api.fetchDevices();
        assertEquals(1, devices.size());
        final MieleDevice dishwasher = devices.iterator().next();
        assertEquals("G7560", dishwasher.getData()
                .getJSONObject("ident")
                .getJSONObject("deviceIdentLabel")
                .get("techType").toString());
    }
}