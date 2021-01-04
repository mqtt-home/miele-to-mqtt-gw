package de.rnd7.mieletomqtt;

import java.io.File;
import java.io.IOException;
import java.time.ZoneId;
import java.util.Optional;
import java.util.concurrent.Executors;
import java.util.concurrent.ScheduledExecutorService;
import java.util.concurrent.TimeUnit;

import de.rnd7.miele.ConfigMiele;
import de.rnd7.miele.ConfigMieleToken;
import de.rnd7.miele.api.SSEClient;
import de.rnd7.mieletomqtt.config.ConfigPersistor;
import de.rnd7.mqtt.GwMqttClient;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import com.google.common.eventbus.EventBus;

import de.rnd7.mieletomqtt.config.Config;
import de.rnd7.mieletomqtt.config.ConfigParser;
import de.rnd7.miele.api.MieleAPI;

public class Main {

    private static final Logger LOGGER = LoggerFactory.getLogger(Main.class);

    @SuppressWarnings("squid:S2189")
    public Main(final Config config, final Optional<File> configFile) {
        LOGGER.debug("Debug enabled");
        LOGGER.info("Info enabled");

        if (configFile.isPresent()) {
            LOGGER.warn("No writable config file available. Login token cannot be persisted.");
        }

        final EventBus eventBus = new EventBus();
        final GwMqttClient mqttClient = new GwMqttClient(config.getMqtt(), eventBus);
        eventBus.register(mqttClient);

        final MieleAPI mieleAPI = new MieleAPI(config.getMiele())
                .setTokenListener(new ConfigPersistor(configFile, config));

        final ScheduledExecutorService executor = Executors.newScheduledThreadPool(2);
        executor.scheduleAtFixedRate(mieleAPI::updateToken, 2, 2, TimeUnit.HOURS);

        final MieleEventHandler eventHandler = new MieleEventHandler(eventBus, config.isDeduplicate());

        try {
            if (config.getMiele().getMode() == ConfigMiele.Mode.sse) {
                LOGGER.info("Using Miele SSE api");
                new SSEClient().start(mieleAPI, eventHandler);
            } else {
                LOGGER.info("Using Miele polling api");
                MielePollingHandler handler = new MielePollingHandler(mieleAPI, eventHandler);
                executor.scheduleAtFixedRate(handler::exec, 0, config.getMqtt().getPollingInterval().getSeconds(), TimeUnit.SECONDS);
            }
        } catch (Exception e) {
            LOGGER.error(e.getMessage(), e);
            executor.shutdown();
            mqttClient.shutdown();
        }
    }

    public static void main(final String[] args) {
        if (args.length != 1) {
            LOGGER.error("Expected configuration file as argument");
            return;
        }

        try {
            final File configFile = new File(args[0]);
            new Main(ConfigParser.parse(configFile), Optional.of(configFile).filter(File::canWrite));
        } catch (final IOException e) {
            LOGGER.error(e.getMessage(), e);
        }
    }
}
