package de.rnd7.miele.api;

import de.rnd7.miele.ConfigMiele;
import de.rnd7.miele.ConfigMieleToken;
import org.junit.jupiter.api.Test;

import java.time.ZoneId;
import java.util.ArrayList;
import java.util.List;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertFalse;
import static org.junit.jupiter.api.Assertions.assertNotEquals;
import static org.junit.jupiter.api.Assertions.assertNotNull;

public class MieleAPITest  {

    @Test
    public void testLogin() throws Exception {
        final MieleAPI api = TestHelper.createAPI();
        assertNotNull(api.getToken(), "API token must be available");
    }

    @Test
    public void test_refresh_token() throws Exception {
        final List<Token> tokens = new ArrayList<>();
        final MieleAPI api = TestHelper.createAPI();
        api.setTokenListener(tokens::add);

        final Token token = api.getToken();
        assertNotNull(api.getToken(), "API token must be available");
        assertEquals(1, tokens.size());

        api.updateToken();

        final Token updated = api.getToken();
        assertNotEquals(token.getAccessToken(), updated.getAccessToken());
        assertNotEquals(token.getRefreshToken(), updated.getRefreshToken());
        assertEquals(2, tokens.size());
        assertEquals(token, tokens.get(0));
        assertEquals(updated, tokens.get(1));
    }

    @Test
    public void test_refresh_token_on_start() throws Exception {
        final MieleAPI api = TestHelper.createAPI();
        final Token token = api.getToken();

        final ConfigMiele config = TestHelper.createConfig();
        final ConfigMieleToken configToken = config.getToken();
        configToken.setRefresh(token.getRefreshToken());
        configToken.setAccess(token.getAccessToken());
        configToken.setValidUntil(token.getExpiresAt().atZone(ZoneId.systemDefault()));

        final MieleAPI nextAPI = new MieleAPI(config);
        final Token nextToken = nextAPI.getToken();
        assertNotEquals(token.getAccessToken(), nextToken.getAccessToken());
        assertNotEquals(token.getRefreshToken(), nextToken.getRefreshToken());
    }

    @Test
    public void test_device_list() throws Exception {
        final MieleAPI api = TestHelper.createAPI();
        final List<MieleDevice> devices = api.fetchDevices();
        assertFalse(devices.isEmpty(), "Expected at least one device");
        final MieleDevice dishwasher = devices.iterator().next();
        assertFalse(dishwasher.getData()
            .getJSONObject("ident")
            .getJSONObject("deviceIdentLabel")
            .get("techType").toString().isEmpty(), "Expected device name to be non-empty");
    }
}
