package de.rnd7.mieletomqtt.miele;

import java.io.File;
import java.io.IOException;
import java.util.Arrays;
import java.util.List;
import java.util.stream.Collectors;

import org.json.JSONObject;

import com.google.api.client.auth.oauth2.AuthorizationCodeFlow;
import com.google.api.client.auth.oauth2.BearerToken;
import com.google.api.client.auth.oauth2.ClientParametersAuthentication;
import com.google.api.client.auth.oauth2.Credential;
import com.google.api.client.extensions.java6.auth.oauth2.AuthorizationCodeInstalledApp;
import com.google.api.client.extensions.jetty.auth.oauth2.LocalServerReceiver;
import com.google.api.client.http.GenericUrl;
import com.google.api.client.http.HttpRequestFactory;
import com.google.api.client.http.HttpTransport;
import com.google.api.client.http.javanet.NetHttpTransport;
import com.google.api.client.json.JsonFactory;
import com.google.api.client.json.JsonObjectParser;
import com.google.api.client.json.jackson2.JacksonFactory;
import com.google.api.client.util.store.FileDataStoreFactory;

public class MieleOAuthAPI {
	private static final String TOKEN_SERVER_URL = "https://api.mcs3.miele.com/thirdparty/token";
	private static final String AUTHORIZATION_SERVER_URL = "https://api.mcs3.miele.com/thirdparty/login";

	/** OAuth 2 scope. */
	private static final String SCOPE = "read";

	/** Global instance of the HTTP transport. */
	private static final HttpTransport HTTP_TRANSPORT = new NetHttpTransport();

	/** Global instance of the JSON factory. */
	private static final JsonFactory JSON_FACTORY = new JacksonFactory();

	/** Port in the "Callback URL". */
	public static final int PORT = 8080;

	/** Domain name in the "Callback URL". */
	public static final String DOMAIN = "127.0.0.1";

	/** Directory to store user credentials. */
	private static final File DATA_STORE_DIR = new File(System.getProperty("user.home"), ".store/miele");
	private final FileDataStoreFactory dataStoreFactory;
	private final HttpRequestFactory requestFactory;

	public MieleOAuthAPI(final String apiKey, final String apiSecret) throws IOException {
		this.dataStoreFactory = new FileDataStoreFactory(DATA_STORE_DIR);
		final Credential credential = this.authorize(apiKey, apiSecret);

		this.requestFactory = HTTP_TRANSPORT.createRequestFactory(request -> {
			credential.initialize(request);
			request.setParser(new JsonObjectParser(JSON_FACTORY));
		});
	}

	private Credential authorize(final String apiKey, final String apiSecret) throws IOException {
		// set up authorization code flow
		final var flow = new AuthorizationCodeFlow.Builder(
				BearerToken.authorizationHeaderAccessMethod(), HTTP_TRANSPORT, JSON_FACTORY,
				new GenericUrl(TOKEN_SERVER_URL), new ClientParametersAuthentication(apiKey, apiSecret), apiKey,
				AUTHORIZATION_SERVER_URL).setScopes(Arrays.asList(SCOPE)).setDataStoreFactory(this.dataStoreFactory)
						.build();
		// authorize
		final var receiver = new LocalServerReceiver.Builder().setHost(DOMAIN).setPort(PORT).build();
		return new AuthorizationCodeInstalledApp(flow, receiver).authorize("user");
	}

	public List<MieleDevice> fetchDevices() throws IOException {
		final var url = new GenericUrl("https://api.mcs3.miele.com/v1/devices");
		final var request = this.requestFactory.buildGetRequest(url);
		final var response = request.execute();
		final var devices = new JSONObject(response.parseAsString());
		return devices.keySet().stream().map(id -> new MieleDevice(id, devices.getJSONObject(id)))
				.collect(Collectors.toList());
	}

}
