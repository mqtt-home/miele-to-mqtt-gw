package de.rnd7.mieletomqtt.miele;

import java.io.IOException;
import java.io.InputStream;
import java.net.HttpURLConnection;
import java.net.URL;
import java.net.URLEncoder;
import java.nio.charset.StandardCharsets;
import java.util.List;
import java.util.regex.Matcher;
import java.util.regex.Pattern;
import java.util.stream.Collectors;

import org.apache.commons.io.IOUtils;
import org.apache.http.Header;
import org.apache.http.client.methods.CloseableHttpResponse;
import org.apache.http.client.methods.HttpPost;
import org.apache.http.entity.StringEntity;
import org.apache.http.impl.client.CloseableHttpClient;
import org.apache.http.impl.client.HttpClients;
import org.apache.http.util.EntityUtils;
import org.json.JSONObject;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import de.rnd7.mieletomqtt.Main;

public class MieleAPI {
	private static final Logger LOGGER = LoggerFactory.getLogger(Main.class);

	private final String clientId;
	private final String clientSecret;
	private final String username;
	private final String password;
	private Token token;

	public MieleAPI(final String clientId, final String clientSecret, final String username, final String password) {
		this.clientId = clientId;
		this.clientSecret = clientSecret;
		this.username = username;
		this.password = password;

		this.updateToken();
	}

	public List<MieleDevice> fetchDevices() throws IOException {
		final var url = new URL("https://api.mcs3.miele.com/v1/devices/");
		final var connection = (HttpURLConnection) url.openConnection();
		connection.setRequestMethod("GET");

		connection.setRequestProperty("Content-Type", "application/json");
		connection.setRequestProperty("Authorization", "Bearer " + this.token.getAccessToken());

		try (var in = connection.getInputStream()) {
			final var devices = new JSONObject(IOUtils.toString(in, StandardCharsets.UTF_8));

			return devices.keySet().stream().map(id -> new MieleDevice(id, devices.getJSONObject(id)))
					.collect(Collectors.toList());
		}
	}

	public void updateToken() {
		try {
			final var code = this.fetchCode();
			this.token = this.fetchToken(code);
		} catch (final IOException e) {
			LOGGER.error("Error while fetching token", e);
		}
	}

	private Token fetchToken(final String code) throws IOException {
		try (var httpclient = HttpClients.createDefault()) {
			final String request = String.format(
					"client_id=%s&client_secret=%s&code=%s&redirect_uri=%s&grant_type=authorization_code&state=token",
					this.clientId, this.clientSecret, code,
					URLEncoder.encode("/v1/devices", StandardCharsets.UTF_8.name()));

			final var post = new HttpPost("https://api.mcs3.miele.com/thirdparty/token");
			post.setHeader("Content-Type", "application/x-www-form-urlencoded");
			post.setEntity(new StringEntity(request));

			try (var response = httpclient.execute(post)) {
				final String page = EntityUtils.toString(response.getEntity(), StandardCharsets.UTF_8);

				return Token.from(new JSONObject(page));
			}
		}
	}

	private String fetchCode() throws IOException {
		try (var httpclient = HttpClients.createDefault()) {
			final String request = String.format(
					"email=%s&password=%s&redirect_uri=%s&state=login&response_type=code&client_id=%s&vgInformationSelector=%s",
					URLEncoder.encode(this.username, StandardCharsets.UTF_8.name()),
					URLEncoder.encode(this.password, StandardCharsets.UTF_8.name()),
					URLEncoder.encode("/v1/", StandardCharsets.UTF_8.name()), this.clientId, "de-de");

			final var post = new HttpPost("https://api.mcs3.miele.com/oauth/auth");
			post.setHeader("Content-Type", "application/x-www-form-urlencoded");
			post.setEntity(new StringEntity(request));

			try (var response = httpclient.execute(post)) {

				final var header = response.getHeaders("Location")[0];
				final var value = header.getValue();

				final var pattern = Pattern.compile("code=([a-z0-9_]+)", Pattern.CASE_INSENSITIVE);
				final var matcher = pattern.matcher(value);

				if (matcher.find()) {
					return matcher.group(1);
				} else {
					return null;
				}
			}
		}
	}
}
