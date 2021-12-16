package de.rnd7.mieletomqtt.miele;

import de.rnd7.miele.ConfigMiele;
import de.rnd7.mieletomqtt.Main;
import de.rnd7.mieletomqtt.config.Config;
import de.rnd7.mqttgateway.Events;
import de.rnd7.mqttgateway.GwMqttClient;
import de.rnd7.mqttgateway.config.ConfigMqtt;
import org.json.JSONObject;
import org.junit.jupiter.api.Assertions;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;
import org.testcontainers.containers.GenericContainer;
import org.testcontainers.containers.wait.strategy.HttpWaitStrategy;
import org.testcontainers.junit.jupiter.Container;
import org.testcontainers.junit.jupiter.Testcontainers;
import org.testcontainers.utility.DockerImageName;

import java.net.URISyntaxException;
import java.time.Duration;
import java.util.Optional;
import java.util.UUID;

import static de.rnd7.mieletomqtt.miele.MqttIntegrationTest.MQTT;
import static de.rnd7.mieletomqtt.miele.MqttIntegrationTest.WEBUI;
import static org.awaitility.Awaitility.await;
import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertNotNull;
import static org.junit.jupiter.api.Assertions.assertTrue;

@Testcontainers
public class EndToEndIntegrationTest {
    @Container
    public GenericContainer<?> activeMQ = new GenericContainer<>(DockerImageName.parse("rmohr/activemq:5.15.9"))
        .withExposedPorts(MQTT, WEBUI)
        .waitingFor(new HttpWaitStrategy().forPort(WEBUI));

    private Config createDefaultConfig() {
        final Config config = new Config();
        config.getMiele()
            .setClientId(TestHelper.forceEnv("MIELE_CLIENT_ID"))
            .setClientSecret(TestHelper.forceEnv("MIELE_CLIENT_SECRET"))
            .setUsername(TestHelper.forceEnv("MIELE_USERNAME"))
            .setPassword(TestHelper.forceEnv("MIELE_PASSWORD"))
            .setPollingInterval(Duration.ofSeconds(2));

        config.getMqtt()
            .setUrl(String.format("tcp://%s:%s", activeMQ.getHost(), activeMQ.getMappedPort(MQTT)))
            .setDefaultTopic("miele");
        return config;
    }

    @Test
    void testPollingEndToEnd() throws URISyntaxException {
        assertConfig(createDefaultConfig());
    }

    @Test
    void testSSEEndToEnd() throws URISyntaxException {
        final Config config = createDefaultConfig();
        config.getMiele().setMode(ConfigMiele.Mode.sse);
        assertConfig(config);
    }

    private void assertConfig(final Config config) throws URISyntaxException {
        final EndToEndMessages messages = start(config);

        assertEquals("online", messages.getState().getRaw());
        assertEquals("connected", messages.getMiele().getRaw());

        final JSONObject smallData = new JSONObject(messages.getSmall().getRaw());
        assertNotNull(smallData.get("phase"));
        assertNotNull(smallData.get("remainingDurationMinutes"));
        assertNotNull(smallData.get("timeCompleted"));
        assertNotNull(smallData.get("remainingDuration"));
        assertNotNull(smallData.get("phaseId"));
        assertNotNull(smallData.get("state"));

        final JSONObject fullData = new JSONObject(messages.getFull().getRaw());

        Assertions.assertFalse(messages.getFull().getRaw().isEmpty());
        assertNotNull(fullData.get("ident"));
    }

    private EndToEndMessages start(final Config config) throws URISyntaxException {
        final EndToEndMessages listener = createMessageListener();

        final Thread thread = new Thread(() -> {
            new Main(config, Optional.empty());
        });

        thread.start();

        // Wait until all messages are there
        await().atMost(Duration.ofSeconds(20)).until(listener::isFulfilled);

        thread.interrupt();

        return listener;
    }

    private void awaitConnected(final GwMqttClient client) {
        await().atMost(Duration.ofSeconds(2)).until(client::isConnected);
        assertTrue(client.isConnected());
    }

    private EndToEndMessages createMessageListener() throws URISyntaxException {
        final ConfigMqtt config = new ConfigMqtt()
            .setUrl(String.format("tcp://%s:%s", activeMQ.getHost(), activeMQ.getMappedPort(MQTT)))
            .setAutoPublish(false);

        final EndToEndMessages messages = new EndToEndMessages();
        Events.register(messages);

        final GwMqttClient client = GwMqttClient.start(config);
        awaitConnected(client);
        client.subscribe("#");

        return messages;
    }
}
