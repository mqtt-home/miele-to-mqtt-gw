package de.rnd7.mieletomqtt.miele;

import de.rnd7.mqttgateway.Events;
import de.rnd7.mqttgateway.GwMqttClient;
import de.rnd7.mqttgateway.Message;
import de.rnd7.mqttgateway.PublishMessage;
import de.rnd7.mqttgateway.config.ConfigMqtt;
import org.junit.jupiter.api.Assertions;
import org.junit.jupiter.api.Test;
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
    void testMqtt() throws Exception {
        final ConfigMqtt configMqtt = new ConfigMqtt()
            .setUrl(String.format("tcp://%s:%s", activeMQ.getHost(), activeMQ.getMappedPort(MQTT)))
            .setTopic("home/miele")
            .setAutoPublish(false);

        final GwMqttClient client = GwMqttClient.start(configMqtt);

        final MessageListener listener = new MessageListener();
        Events.register(listener);
        client.subscribe("home/miele/#");

        Events.post(PublishMessage.relative("hi/there", "message"));

        await().atMost(Duration.ofSeconds(10)).until(() -> !listener.getMessages().isEmpty());

        Assertions.assertEquals(1, listener.getMessages().size());
        Message message = listener.getMessages().iterator().next();
        Assertions.assertEquals("home/miele/hi/there", message.getTopic());
        Assertions.assertEquals("message", message.getRaw());
    }
}
