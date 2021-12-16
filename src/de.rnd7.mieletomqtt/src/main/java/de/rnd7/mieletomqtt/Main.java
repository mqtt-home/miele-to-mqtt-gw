package de.rnd7.mieletomqtt;

import java.io.File;
import java.io.IOException;
import java.util.Optional;
import java.util.concurrent.Executors;
import java.util.concurrent.ScheduledExecutorService;
import java.util.concurrent.TimeUnit;

import de.rnd7.miele.ConfigMiele;
import de.rnd7.miele.api.MieleAPIState;
import de.rnd7.miele.api.SSEClient;
import de.rnd7.mieletomqtt.config.ConfigPersistor;
import de.rnd7.mqttgateway.Events;
import de.rnd7.mqttgateway.GwMqttClient;
import de.rnd7.mqttgateway.PublishMessage;
import de.rnd7.mqttgateway.config.ConfigParser;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import de.rnd7.mieletomqtt.config.Config;
import de.rnd7.miele.api.MieleAPI;

import static de.rnd7.miele.api.MieleEventListener.MIELE_STATE;

public class Main {

    private static final Logger LOGGER = LoggerFactory.getLogger(Main.class);

    @SuppressWarnings("squid:S2189")
    public Main(final Config config, final Optional<File> configFile) {
        LOGGER.debug("Debug enabled");
        LOGGER.info("Info enabled");

        if (!configFile.isPresent()) {
            LOGGER.warn("No writable config file available. Login token cannot be persisted.");
        }

        registerOfflineHook();
        final ScheduledExecutorService executor = Executors.newScheduledThreadPool(2);

        try {
            GwMqttClient.start(config.getMqtt()
                .setDefaultTopic("miele"));

            final MieleAPI mieleAPI = new MieleAPI(config.getMiele())
                .setTokenListener(new ConfigPersistor(configFile, config))
                .login();

            executor.scheduleAtFixedRate(() -> {
                try {
                    mieleAPI.updateToken();
                } catch (IOException e) {
                    LOGGER.error("Error updating token: {}", e.getMessage(), e);
                }
            }, 2, 2, TimeUnit.HOURS);

            final MieleEventHandler eventHandler = new MieleEventHandler();
            if (config.getMiele().getMode() == ConfigMiele.Mode.sse) {
                LOGGER.info("Using Miele SSE api");
                new SSEClient()
                    .start(mieleAPI, eventHandler);
            } else {
                LOGGER.info("Using Miele polling api");
                new MielePollingHandler(mieleAPI, eventHandler)
                    .start(config.getMiele().getPollingInterval().getSeconds());
            }
        } catch (Exception e) {
            LOGGER.error(e.getMessage(), e);
        }
        finally {
            executor.shutdown();
            disconnected();
        }
    }

    private void registerOfflineHook() {
        Runtime.getRuntime().addShutdownHook(new Thread() {
            @Override
            public void run() {
                disconnected();
            }
        });
    }

    private void disconnected() {
        Events.post(PublishMessage.relative(MIELE_STATE, MieleAPIState.disconnected.toString()));
    }

    public static void main(final String[] args) {
        if (args.length != 1) {
            LOGGER.error("Expected configuration file as argument");
            return;
        }

        try {
            final File configFile = new File(args[0]);
            new Main(ConfigParser.parse(configFile, Config.class),
                Optional.of(configFile).filter(File::canWrite));
        } catch (final IOException e) {
            LOGGER.error(e.getMessage(), e);
        }

        System.exit(1);
    }
}
