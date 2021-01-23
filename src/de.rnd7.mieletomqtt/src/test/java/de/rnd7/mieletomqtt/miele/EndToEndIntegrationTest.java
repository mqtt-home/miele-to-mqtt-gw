package de.rnd7.mieletomqtt.miele;

import de.rnd7.miele.ConfigMiele;
import de.rnd7.mieletomqtt.Main;
import de.rnd7.mieletomqtt.config.Config;
import de.rnd7.mqttgateway.Events;
import de.rnd7.mqttgateway.GwMqttClient;
import de.rnd7.mqttgateway.config.ConfigMqtt;
import org.json.JSONObject;
import org.junit.Assert;
import org.junit.Test;
import org.testcontainers.containers.GenericContainer;
import org.testcontainers.containers.wait.strategy.HttpWaitStrategy;
import org.testcontainers.junit.jupiter.Container;
import org.testcontainers.junit.jupiter.Testcontainers;
import org.testcontainers.utility.DockerImageName;

import java.time.Duration;
import java.util.Optional;
import java.util.UUID;

import static de.rnd7.mieletomqtt.miele.MqttIntegrationTest.MQTT;
import static de.rnd7.mieletomqtt.miele.MqttIntegrationTest.WEBUI;
import static org.awaitility.Awaitility.await;

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
            .setClientId(UUID.randomUUID().toString());
        return config;
    }

    @Test
    public void testPollingEndToEnd() {
        assertConfig(createDefaultConfig());
    }

    @Test
    public void testSSEEndToEnd() {
        final Config config = createDefaultConfig();
        config.getMiele().setMode(ConfigMiele.Mode.sse);
        assertConfig(config);
    }

    private void assertConfig(final Config config) {
        final EndToEndMessages messages = start(config);

        Assert.assertEquals("online", messages.getState().getRaw());
        Assert.assertEquals("connected", messages.getMiele().getRaw());

        final JSONObject smallData = new JSONObject(messages.getSmall().getRaw());
        Assert.assertNotNull(smallData.get("phase"));
        Assert.assertNotNull(smallData.get("remainingDurationMinutes"));
        Assert.assertNotNull(smallData.get("timeCompleted"));
        Assert.assertNotNull(smallData.get("remainingDuration"));
        Assert.assertNotNull(smallData.get("phaseId"));
        Assert.assertNotNull(smallData.get("state"));

        final JSONObject fullData = new JSONObject(messages.getFull().getRaw());

        Assert.assertFalse(messages.getFull().getRaw().isEmpty());
        Assert.assertNotNull(fullData.get("ident"));
    }

    private EndToEndMessages start(Config config) {
        final EndToEndMessages listener = createMessageListener();

        final Thread thread = new Thread(() -> {
            new Main(config, Optional.empty());
        });

        thread.start();

        // Wait for at least two messages
        await().atMost(Duration.ofSeconds(10)).until(listener::isFulfilled);

        thread.interrupt();

        return listener;
    }

    private EndToEndMessages createMessageListener() {
        final ConfigMqtt config = new ConfigMqtt()
            .setUrl(String.format("tcp://%s:%s", activeMQ.getHost(), activeMQ.getMappedPort(MQTT)));

        final GwMqttClient client = GwMqttClient.start(config);

        final EndToEndMessages messages = new EndToEndMessages();
        Events.register(messages);
        Events.register(client);
        client.subscribe("miele/#");
        return messages;
    }
}
