package de.rnd7.miele.api;

import java.io.IOException;
import java.io.InputStream;
import java.net.HttpURLConnection;
import java.net.URL;
import java.net.URLEncoder;
import java.nio.charset.StandardCharsets;
import java.time.LocalDateTime;
import java.util.Collections;
import java.util.List;
import java.util.regex.Matcher;
import java.util.regex.Pattern;
import java.util.stream.Collectors;

import de.rnd7.miele.ConfigMiele;
import de.rnd7.miele.ConfigMieleToken;
import org.apache.commons.io.IOUtils;
import org.apache.http.Header;
import org.apache.http.StatusLine;
import org.apache.http.client.methods.CloseableHttpResponse;
import org.apache.http.client.methods.HttpPost;
import org.apache.http.entity.StringEntity;
import org.apache.http.impl.client.CloseableHttpClient;
import org.apache.http.impl.client.HttpClients;
import org.apache.http.util.EntityUtils;
import org.json.JSONObject;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

public class MieleAPI {
    private static final Logger LOGGER = LoggerFactory.getLogger(MieleAPI.class);

    public static final int MAX_RECONNECT_RETRY = 10;
    public static final int RECONNECT_SLEEP_MS = 120_000;

    private final String clientId;
    private final String clientSecret;
    private final String username;
    private final String password;
    private Token token;

    private TokenListener tokenListener;

    public MieleAPI(final ConfigMiele config) {
        this.clientId = config.getClientId();
        this.clientSecret = config.getClientSecret();
        this.username = config.getUsername();
        this.password = config.getPassword();

        final ConfigMieleToken ctoken = config.getToken();
        if (ctoken != null && ctoken.isValid()) {
            refreshToken(new Token()
                .setAccessToken(ctoken.getAccess())
                .setRefreshToken(ctoken.getRefresh())
                .setExpiresAt(ctoken.getValidUntil().toLocalDateTime()));
        }
    }

    public MieleAPI setTokenListener(final TokenListener tokenListener) {
        this.tokenListener = tokenListener;
        return this;
    }

    public Token getToken() throws IOException {
        if (token == null) {
            updateToken();
        }

        return token;
    }

    public List<MieleDevice> fetchDevices() throws IOException {
        if (this.token == null) {
            LOGGER.error("No token available");
            updateToken();
            if (token == null) {
                LOGGER.error("No token available (give up)");
                return Collections.emptyList();
            }
        }

        final URL url = new URL("https://api.mcs3.miele.com/v1/devices/");
        final HttpURLConnection connection = (HttpURLConnection) url.openConnection();
        connection.setRequestMethod("GET");

        connection.setRequestProperty("Content-Type", "application/json");
        connection.setRequestProperty("Authorization", "Bearer " + this.token.getAccessToken());

        try (InputStream in = connection.getInputStream()) {
            final JSONObject devices = new JSONObject(IOUtils.toString(in, StandardCharsets.UTF_8));

            return devices.keySet().stream().map(id -> new MieleDevice(id, devices.getJSONObject(id)))
                .collect(Collectors.toList());
        }
    }

    private boolean refreshToken(final Token toBeRefreshed) {
        if (toBeRefreshed != null && toBeRefreshed.getExpiresAt().isAfter(LocalDateTime.now())) {
            try {
                setToken(refreshToken(toBeRefreshed.getRefreshToken()));
                return true;
            } catch (IOException e) {
                LOGGER.error("Cannot refresh token", e);
            }
        }

        return false;
    }

    public void updateToken() throws IOException {
        try {
            if (refreshToken(this.token)) {
                return;
            }

            final String code = this.fetchCode();
            setToken(this.fetchToken(code));
        } catch (final IOException e) {
            throw new IOException("Error while fetching token", e);
        }
    }

    public void setToken(final Token token) {
        this.token = token;

        if (tokenListener != null) {
            tokenListener.acceptToken(this.token);
        }
    }

    private Token fetchToken(final String code) throws IOException {
        try (CloseableHttpClient httpclient = HttpClients.createDefault()) {
            final String request = String.format(
                "client_id=%s&client_secret=%s&code=%s&redirect_uri=%s&grant_type=authorization_code&state=token",
                this.clientId, this.clientSecret, code,
                URLEncoder.encode("/v1/devices", StandardCharsets.UTF_8.name()));

            final HttpPost post = new HttpPost("https://api.mcs3.miele.com/thirdparty/token");
            post.setHeader("Content-Type", "application/x-www-form-urlencoded");
            post.setEntity(new StringEntity(request));

            try (final CloseableHttpResponse response = httpclient.execute(post)) {
                final String page = EntityUtils.toString(response.getEntity(), StandardCharsets.UTF_8);

                return Token.from(new JSONObject(page));
            }
        }
    }

    private Token refreshToken(final String refreshToken) throws IOException {
        try (CloseableHttpClient httpclient = HttpClients.createDefault()) {
            final String request = String.format(
                "client_id=%s&client_secret=%s&refresh_token=%s&grant_type=refresh_token",
                this.clientId, this.clientSecret, refreshToken);

            final HttpPost post = new HttpPost("https://api.mcs3.miele.com/thirdparty/token");
            post.setHeader("Content-Type", "application/x-www-form-urlencoded");
            post.setEntity(new StringEntity(request));

            try (final CloseableHttpResponse response = httpclient.execute(post)) {
                final String page = EntityUtils.toString(response.getEntity(), StandardCharsets.UTF_8);
                final int statusCode = response.getStatusLine().getStatusCode();
                if (statusCode == 401) {
                    throw new IOException("401 - " + getMessage(page));
                } else if (statusCode == 200) {
                    return Token.from(new JSONObject(page));
                } else {
                    throw new IOException("Unexpected status code while refreshing token: " + statusCode);
                }
            }
        }
    }

    private String getMessage(final String page) {
        try {
            return new JSONObject(page).getString("message");
        } catch (Exception e) {
            LOGGER.trace(e.getMessage(), e);
        }

        return "no message";
    }

    private String fetchCode() throws IOException {
        try (final CloseableHttpClient httpclient = HttpClients.createDefault()) {
            final String request = String.format(
                "email=%s&password=%s&redirect_uri=%s&state=login&response_type=code&client_id=%s&vgInformationSelector=%s",
                URLEncoder.encode(this.username, StandardCharsets.UTF_8.name()),
                URLEncoder.encode(this.password, StandardCharsets.UTF_8.name()),
                URLEncoder.encode("/v1/", StandardCharsets.UTF_8.name()), this.clientId, "de-de");

            final HttpPost post = new HttpPost("https://api.mcs3.miele.com/oauth/auth");
            post.setHeader("Content-Type", "application/x-www-form-urlencoded");
            post.setEntity(new StringEntity(request));

            try (final CloseableHttpResponse response = httpclient.execute(post)) {
                final StatusLine statusLine = response.getStatusLine();
                final int code = statusLine.getStatusCode();
                if (code != 302) {
                    throw new IOException(String.format("Unexpected code: %s during login (%s).", code, statusLine.getReasonPhrase()));
                }

                final Header[] locations = response.getHeaders("Location");
                if (locations.length == 0) {
                    throw new IOException("Error during login (fetch code), location header was expected to be set. Please check your credentials.");
                }

                final Header header = locations[0];
                final String value = header.getValue();

                final Pattern pattern = Pattern.compile("code=([a-z0-9_]+)", Pattern.CASE_INSENSITIVE);
                final Matcher matcher = pattern.matcher(value);

                if (matcher.find()) {
                    return matcher.group(1);
                } else {
                    throw new IOException("Error during login (fetch code), no code found.");
                }
            }
        }
    }

    public boolean waitReconnect() {
        try {
            for (int i = 1; i <= MAX_RECONNECT_RETRY ; i++) {
                LOGGER.info("Wait for the network connection to come back (retry {}/{})", i, MAX_RECONNECT_RETRY);

                // Wait some time (e.g. short interruption of the internet connection)
                Thread.sleep(RECONNECT_SLEEP_MS);

                if (tryReconnect()) {
                    return true;
                }
            }
        } catch (InterruptedException e) {
            LOGGER.debug(e.getMessage(), e);
            Thread.currentThread().interrupt();
        }

        LOGGER.info("Cannot connect to the Miele API (giving up)");
        return false;
    }

    private boolean tryReconnect() {
        try {
            updateToken();
            return true;
        } catch (IOException e) {
            if (LOGGER.isDebugEnabled()) {
                LOGGER.debug(e.getMessage(), e);
            }
            else {
                LOGGER.info(e.getMessage());
            }
        }
        return false;
    }
}
