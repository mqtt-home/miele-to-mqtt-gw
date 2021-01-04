package de.rnd7.mieletomqtt.miele;

import com.google.common.eventbus.EventBus;
import de.rnd7.miele.ConfigMiele;
import de.rnd7.mieletomqtt.Main;
import de.rnd7.mieletomqtt.config.Config;
import de.rnd7.mqtt.ConfigMqtt;
import de.rnd7.mqtt.GwMqttClient;
import de.rnd7.mqtt.ReceivedMessage;
import org.awaitility.Duration;
import org.json.JSONObject;
import org.junit.Assert;
import org.junit.Rule;
import org.junit.Test;
import org.testcontainers.containers.GenericContainer;
import org.testcontainers.containers.wait.strategy.HttpWaitStrategy;
import org.testcontainers.utility.DockerImageName;

import java.io.File;
import java.util.List;
import java.util.Optional;
import java.util.UUID;
import java.util.function.Function;

import static de.rnd7.mieletomqtt.miele.MqttIntegrationTest.MQTT;
import static de.rnd7.mieletomqtt.miele.MqttIntegrationTest.WEBUI;
import static org.awaitility.Awaitility.await;

public class EndToEndIntegrationTest {
    @Rule
    public GenericContainer activeMQ = new GenericContainer(DockerImageName.parse("rmohr/activemq:5.15.9"))
        .withExposedPorts(MQTT, WEBUI)
        .waitingFor(new HttpWaitStrategy().forPort(WEBUI));

    private Config createDefaultConfig() {
        final Config config = new Config();
        config.getMiele()
            .setClientId(TestHelper.forceEnv("MIELE_CLIENT_ID"))
            .setClientSecret(TestHelper.forceEnv("MIELE_CLIENT_SECRET"))
            .setUsername(TestHelper.forceEnv("MIELE_USERNAME"))
            .setPassword(TestHelper.forceEnv("MIELE_PASSWORD"));

        config.getMqtt()
            .setPollingInterval(java.time.Duration.ofSeconds(2))
            .setBroker(activeMQ.getHost(), activeMQ.getMappedPort(MQTT))
            .setClientId(UUID.randomUUID().toString());
        return config;
    }

    private List<ReceivedMessage> start(Config config, Function<Integer, Boolean> endCondition) {
        final MessageListener listener = createMessageListener();

        final Thread thread = new Thread(() -> {
            new Main(config, Optional.empty());
        });

        thread.start();

        // Wait for at least two messages
        await().atMost(Duration.TEN_SECONDS).until(() -> endCondition.apply(listener.getMessages().size()));

        thread.interrupt();

        return listener.getMessages();
    }

    @Test
    public void testPollingEndToEnd() {
        final Config config = createDefaultConfig();

        final List<ReceivedMessage> messages = start(config, size -> size > 1);

        final ReceivedMessage full = getFullMessage(messages);
        final ReceivedMessage small = getSmallMessage(messages);

        final JSONObject smallData = new JSONObject(small.getData());
        Assert.assertNotNull(smallData.get("phase"));
        Assert.assertNotNull(smallData.get("remainingDurationMinutes"));
        Assert.assertNotNull(smallData.get("timeCompleted"));
        Assert.assertNotNull(smallData.get("remainingDuration"));
        Assert.assertNotNull(smallData.get("phaseId"));
        Assert.assertNotNull(smallData.get("state"));

        final JSONObject fullData = new JSONObject(full.getData());

        Assert.assertFalse(full.getData().isEmpty());
        Assert.assertNotNull(fullData.get("ident"));
    }

    @Test
    public void testSSEEndToEnd() {
        final Config config = createDefaultConfig();
        config.getMiele().setMode(ConfigMiele.Mode.sse);

        final List<ReceivedMessage> messages = start(config, size -> size > 1);

        final ReceivedMessage full = getFullMessage(messages);
        final ReceivedMessage small = getSmallMessage(messages);

        final JSONObject smallData = new JSONObject(small.getData());
        Assert.assertNotNull(smallData.get("phase"));
        Assert.assertNotNull(smallData.get("remainingDurationMinutes"));
        Assert.assertNotNull(smallData.get("timeCompleted"));
        Assert.assertNotNull(smallData.get("remainingDuration"));
        Assert.assertNotNull(smallData.get("phaseId"));
        Assert.assertNotNull(smallData.get("state"));

        final JSONObject fullData = new JSONObject(full.getData());

        Assert.assertFalse(full.getData().isEmpty());
        Assert.assertNotNull(fullData.get("ident"));
    }

    private ReceivedMessage getSmallMessage(final List<ReceivedMessage> messages) {
        return messages.stream().filter(m -> !m.getTopic().endsWith("/full")).findFirst()
            .orElseThrow(() -> new IllegalStateException("Expected at least one small message to be present."));
    }

    private ReceivedMessage getFullMessage(final List<ReceivedMessage> messages) {
        return messages.stream().filter(m -> m.getTopic().endsWith("/full")).findFirst()
            .orElseThrow(() -> new IllegalStateException("Expected at least one full message to be present."));
    }

    private MessageListener createMessageListener() {
        final EventBus eventBus = new EventBus();
        final GwMqttClient client = new GwMqttClient(
            new ConfigMqtt().setBroker(activeMQ.getHost(), activeMQ.getMappedPort(MQTT)), eventBus);

        final MessageListener listener = new MessageListener();
        eventBus.register(listener);
        eventBus.register(client);
        client.subscribe("miele/#");
        return listener;
    }
}
