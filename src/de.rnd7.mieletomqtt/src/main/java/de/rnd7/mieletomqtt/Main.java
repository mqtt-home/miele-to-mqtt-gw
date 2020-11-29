package de.rnd7.mieletomqtt;

import java.io.File;
import java.io.IOException;
import java.util.concurrent.Executors;
import java.util.concurrent.TimeUnit;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import com.google.common.eventbus.EventBus;

import de.rnd7.mieletomqtt.config.Config;
import de.rnd7.mieletomqtt.config.ConfigParser;
import de.rnd7.mieletomqtt.miele.MieleAPI;
import de.rnd7.mieletomqtt.mqtt.GwMqttClient;

public class Main {

	private static final Logger LOGGER = LoggerFactory.getLogger(Main.class);

	private final EventBus eventBus = new EventBus();

	private final MieleAPI mieleAPI;

	@SuppressWarnings("squid:S2189")
	public Main(final Config config) {
		this.eventBus.register(new GwMqttClient(config));

		final var miele = config.getMiele();
		this.mieleAPI = new MieleAPI(miele.getClientId(), miele.getClientSecret(),
				miele.getUsername(), miele.getPassword());

		try {
			final var executor = Executors.newScheduledThreadPool(2);
			executor.scheduleAtFixedRate(this::exec, 0, config.getMqtt().getPollingInterval().getSeconds(), TimeUnit.SECONDS);
			executor.scheduleAtFixedRate(this.mieleAPI::updateToken, 2, 2, TimeUnit.HOURS);

			while (true) {
				Thread.sleep(100);
			}
		} catch (final Exception e) {
			LOGGER.error(e.getMessage(), e);
		}
	}

	private void exec() {
		try {
			for (final var device : this.mieleAPI.fetchDevices()) {
				this.eventBus.post(device.toFullMessage());
				this.eventBus.post(device.toSmallMessage());
			}
		} catch (final Exception e) {
			LOGGER.error(e.getMessage(), e);
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
