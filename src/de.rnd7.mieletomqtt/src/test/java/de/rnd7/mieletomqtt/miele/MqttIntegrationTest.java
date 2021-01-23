package de.rnd7.mieletomqtt.miele;

import com.google.common.eventbus.EventBus;
import de.rnd7.mqttgateway.Events;
import de.rnd7.mqttgateway.GwMqttClient;
import de.rnd7.mqttgateway.Message;
import de.rnd7.mqttgateway.PublishMessage;
import de.rnd7.mqttgateway.config.ConfigMqtt;
import org.junit.Assert;
import org.junit.Rule;
import org.junit.Test;
import org.junit.jupiter.api.BeforeEach;
import org.testcontainers.containers.GenericContainer;
import org.testcontainers.containers.wait.strategy.HttpWaitStrategy;
import org.testcontainers.junit.jupiter.Container;
import org.testcontainers.junit.jupiter.Testcontainers;
import org.testcontainers.utility.DockerImageName;

import java.time.Duration;

import static org.awaitility.Awaitility.await;

@Testcontainers
public class MqttIntegrationTest {
    public static final int MQTT = 1883;
    public static final int WEBUI = 8161;

    @Container
    public GenericContainer<?> activeMQ = new GenericContainer<>(DockerImageName.parse("rmohr/activemq:5.15.9"))
        .withExposedPorts(MQTT, WEBUI)
        .waitingFor(new HttpWaitStrategy().forPort(WEBUI));

    @Test
    public void testMqtt() throws Exception {
        final ConfigMqtt configMqtt = new ConfigMqtt()
            .setUrl(String.format("tcp://%s:%s", activeMQ.getHost(), activeMQ.getMappedPort(MQTT)))
            .setTopic("home/miele");

        final GwMqttClient client = GwMqttClient.start(configMqtt);

        final MessageListener listener = new MessageListener();
        Events.register(listener);
        Events.register(client);
        client.subscribe("home/miele/#");

        Events.post(PublishMessage.absolute("hi/there", "message"));

        await().atMost(Duration.ofSeconds(10)).until(() -> !listener.getMessages().isEmpty());

        Assert.assertEquals(1, listener.getMessages().size());
        Message message = listener.getMessages().iterator().next();
        Assert.assertEquals("home/miele/hi/there", message.getTopic());
        Assert.assertEquals("message", message.getRaw());
    }
}
