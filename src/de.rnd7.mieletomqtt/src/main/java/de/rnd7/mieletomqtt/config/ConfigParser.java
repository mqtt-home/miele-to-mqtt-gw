package de.rnd7.mieletomqtt.config;

import java.io.File;
import java.io.FileInputStream;
import java.io.IOException;
import java.io.InputStream;
import java.nio.charset.StandardCharsets;
import java.time.Duration;
import java.time.ZonedDateTime;

import com.google.gson.Gson;
import com.google.gson.GsonBuilder;
import org.apache.commons.io.IOUtils;

public class ConfigParser {

    private ConfigParser() {
    }

    public static Config parse(final File file) throws IOException {
        try (FileInputStream in = new FileInputStream(file)) {
            return parse(in);
        }
    }

    public static Config parse(final InputStream in) throws IOException {
        final String json = IOUtils.toString(in, StandardCharsets.UTF_8);

        final Gson gson = new GsonBuilder()
            .registerTypeAdapter(Duration.class, new DurationAdapter())
            .registerTypeAdapter(ZonedDateTime.class, new ZonedDateTimeAdapter())
            .create();

        return gson.fromJson(json, Config.class);
    }
}
