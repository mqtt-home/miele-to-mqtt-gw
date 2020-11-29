package de.rnd7.mieletomqtt.config;

import java.io.File;
import java.io.FileInputStream;
import java.io.IOException;
import java.io.InputStream;
import java.nio.charset.StandardCharsets;
import java.time.Duration;

import com.google.gson.Gson;
import com.google.gson.GsonBuilder;
import org.apache.commons.io.IOUtils;
import org.json.JSONObject;

public class ConfigParser {

	private ConfigParser() {
	}

	public static Config parse(final File file) throws IOException {
		try (var in = new FileInputStream(file)) {
			return parse(in);
		}
	}

	public static Config parse(final InputStream in) throws IOException {
		final String json = IOUtils.toString(in, StandardCharsets.UTF_8);

		final var gson = new GsonBuilder()
				.registerTypeAdapter(Duration.class, new DurationDeserializer())
				.create();

		return gson.fromJson(json, Config.class);
	}
}
