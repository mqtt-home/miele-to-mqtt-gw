package de.rnd7.mieletomqtt;

import java.io.File;
import java.io.IOException;
import java.util.concurrent.Executors;
import java.util.concurrent.ScheduledExecutorService;
import java.util.concurrent.TimeUnit;

import de.rnd7.miele.ConfigMiele;
import de.rnd7.miele.api.SSEClient;
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
	public Main(final Config config) {
		LOGGER.debug("Debug enabled");
		LOGGER.info("Info enabled");

		final EventBus eventBus = new EventBus();
		eventBus.register(new GwMqttClient(config.getMqtt()));

		final ConfigMiele miele = config.getMiele();
		MieleAPI mieleAPI = new MieleAPI(miele.getClientId(), miele.getClientSecret(),
				miele.getUsername(), miele.getPassword());

		final ScheduledExecutorService executor = Executors.newScheduledThreadPool(2);
		executor.scheduleAtFixedRate(mieleAPI::updateToken, 2, 2, TimeUnit.HOURS);

		MieleEventHandler eventHandler = new MieleEventHandler(eventBus, config.isDeduplicate());

		if (config.getMiele().getMode() == ConfigMiele.Mode.sse) {
			LOGGER.info("Using Miele SSE api");
			final SSEClient client = new SSEClient();
			client.start(mieleAPI, eventHandler);
		}
		else {
			LOGGER.info("Using Miele polling api");
			MielePollingHandler handler = new MielePollingHandler(mieleAPI, eventHandler);
			executor.scheduleAtFixedRate(handler::exec, 0, config.getMqtt().getPollingInterval().getSeconds(), TimeUnit.SECONDS);
		}
	}

	public static void main(final String[] args) {
		if (args.length != 1) {
			LOGGER.error("Expected configuration file as argument");
			return;
		}

		try {
			new Main(ConfigParser.parse(new File(args[0])));
		} catch (final IOException e) {
			LOGGER.error(e.getMessage(), e);
		}
	}
}
