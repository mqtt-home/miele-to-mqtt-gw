package de.rnd7.mieletomqtt.config;

import com.google.gson.Gson;
import com.google.gson.GsonBuilder;
import de.rnd7.miele.ConfigMieleToken;
import de.rnd7.miele.api.Token;
import de.rnd7.miele.api.TokenListener;
import org.apache.commons.io.FileUtils;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.io.File;
import java.io.IOException;
import java.nio.charset.StandardCharsets;
import java.time.Duration;
import java.time.ZoneId;
import java.time.ZonedDateTime;
import java.util.Optional;

public class ConfigPersistor implements TokenListener {
    private static final Logger LOGGER = LoggerFactory.getLogger(ConfigPersistor.class);

    private final Optional<File> file;
    private final Config config;

    public ConfigPersistor(final Optional<File> file, final Config config) {
        this.file = file;
        this.config = config;
    }

    private void persistToken() {
        final Gson gson = new GsonBuilder()
            .registerTypeAdapter(Duration.class, new DurationAdapter())
            .registerTypeAdapter(ZonedDateTime.class, new ZonedDateTimeAdapter())
            .setPrettyPrinting()
            .create();

        file.ifPresent(f -> {
            try {
                FileUtils.writeStringToFile(f, gson.toJson(config), StandardCharsets.UTF_8);
            } catch (IOException e) {
                LOGGER.error("Error persisting login token: " + e.getMessage(), e);
            }
        });
    }

    @Override
    public void acceptToken(final Token token) {
        final ConfigMieleToken configMieleToken = new ConfigMieleToken();
        configMieleToken.setAccess(token.getAccessToken());
        configMieleToken.setRefresh(token.getRefreshToken());
        configMieleToken.setValidUntil(token.getExpiresAt().atZone(ZoneId.systemDefault()));
        config.getMiele().setToken(configMieleToken);

        persistToken();
    }
}
