package de.rnd7.mqtt;

import java.util.Optional;

import com.google.common.eventbus.EventBus;
import org.eclipse.paho.client.mqttv3.*;
import org.eclipse.paho.client.mqttv3.persist.MemoryPersistence;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import com.google.common.eventbus.Subscribe;

public class GwMqttClient {
	private static final int QOS = 2;
	private static final String CLIENT_ID = "miele-mqtt-gw";

	private static final Logger LOGGER = LoggerFactory.getLogger(GwMqttClient.class);

	private final MemoryPersistence persistence = new MemoryPersistence();
	private final Object mutex = new Object();
	private final ConfigMqtt config;
	private EventBus eventBus;

	private Optional<MqttClient> client;

	public
	GwMqttClient(final ConfigMqtt config, EventBus eventBus) {
		this.config = config;
		this.eventBus = eventBus;
		this.client = this.connect();
	}

	public void subscribe(String topic) {
		final MqttClient mqtt = client.orElseThrow(() -> new IllegalStateException("Cannot subscribe, no client available"));
		try {
			mqtt.subscribe(topic);
		} catch (MqttException e) {
			throw new RuntimeException(e);
		}
	}

	private Optional<MqttClient> connect() {
		try {
			LOGGER.info("Connecting MQTT client");
			final MqttClient result = new MqttClient(this.config.getUrl(),
					this.config.getClientId().orElse(CLIENT_ID),
					this.persistence);

			result.setCallback(new MqttCallback() {
				@Override
				public void connectionLost(Throwable cause) {
					LOGGER.error(cause.getMessage(), cause);
				}

				@Override
				public void messageArrived(String topic, MqttMessage message) throws Exception {
					eventBus.post(new ReceivedMessage(topic, new String(message.getPayload())));
				}

				@Override
				public void deliveryComplete(IMqttDeliveryToken token) {
					// do nothing
				}
			});

			final MqttConnectOptions connOpts = new MqttConnectOptions();
			connOpts.setCleanSession(true);
			config.getUsername().ifPresent(connOpts::setUserName);
			config.getPassword().map(String::toCharArray).ifPresent(connOpts::setPassword);

			result.connect(connOpts);
			LOGGER.info("MQTT client connected");
			return Optional.of(result);
		} catch (final MqttException e) {
			if (LOGGER.isDebugEnabled()) {
				LOGGER.debug(e.getMessage(), e);
			}
			else {
				LOGGER.error(e.getMessage() + " " + Optional.ofNullable(e.getCause()).map(Throwable::getMessage).orElse("No cause."));
			}

			return Optional.empty();
		}
	}

	private void publish(final String topic, final String value) {
		synchronized (this.mutex) {
			LOGGER.debug("publishing {} = {}", topic, value);

			if (!this.client.filter(MqttClient::isConnected).isPresent()) {
				this.client = this.connect();
			}

			this.client.ifPresent(mqttClient -> {
				try {
					final MqttMessage message = new MqttMessage(value.getBytes());
					message.setQos(QOS);
					message.setRetained(config.isRetain());
					mqttClient.publish(topic, message);
				} catch (final MqttException e) {
					LOGGER.error(e.getMessage(), e);
				}
			});
		}
	}

	@Subscribe
	public void publish(final Message message) {
		final String topic = this.config.getFullMessageTopic() + "/" + message.getTopic();
		final String valueString = message.getData();
		this.publish(topic, valueString);
	}

    public void shutdown() {
		client.ifPresent(c -> {
			try {
				c.disconnect();
				c.close();
			} catch (MqttException e) {
				LOGGER.debug(e.getMessage(), e);
			}
		});
    }
}
