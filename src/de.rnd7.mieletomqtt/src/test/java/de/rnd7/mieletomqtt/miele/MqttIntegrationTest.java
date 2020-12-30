package de.rnd7.mieletomqtt.miele;

import com.google.common.eventbus.EventBus;
import com.google.common.eventbus.Subscribe;
import de.rnd7.mqtt.ConfigMqtt;
import de.rnd7.mqtt.GwMqttClient;
import de.rnd7.mqtt.Message;
import de.rnd7.mqtt.ReceivedMessage;
import org.awaitility.Duration;
import org.junit.Assert;
import org.junit.Rule;
import org.junit.Test;
import org.testcontainers.containers.GenericContainer;
import org.testcontainers.containers.wait.strategy.HttpWaitStrategy;
import org.testcontainers.utility.DockerImageName;

import java.util.ArrayList;
import java.util.List;

import static org.awaitility.Awaitility.await;

public class MqttIntegrationTest {
    public static final int MQTT = 1883;
    public static final int WEBUI = 8161;

    public static class MessageListener {
        private List<ReceivedMessage> messages = new ArrayList<>();

        @Subscribe
        public void onMessage(ReceivedMessage message) {
            messages.add(message);
        }
    }

    @Rule
    public GenericContainer activeMQ = new GenericContainer(DockerImageName.parse("rmohr/activemq:5.15.9"))
            .withExposedPorts(MQTT, WEBUI)
            .waitingFor(new HttpWaitStrategy().forPort(WEBUI));

    @Test
    public void testMqtt() throws Exception {
        final EventBus eventBus = new EventBus();
        final GwMqttClient client = new GwMqttClient(
                ConfigMqtt.createFor(activeMQ.getHost(), activeMQ.getMappedPort(MQTT), "home/miele"), eventBus);

        final MessageListener listener = new MessageListener();
        eventBus.register(listener);
        eventBus.register(client);
        client.subscribe("home/miele/#");

        eventBus.post(new Message("hi/there", "message"));

        await().atMost(Duration.TEN_SECONDS).until(() -> !listener.messages.isEmpty());

        Assert.assertEquals(1, listener.messages.size());
        ReceivedMessage message = listener.messages.iterator().next();
        Assert.assertEquals("home/miele/hi/there", message.getTopic());
        Assert.assertEquals("message", message.getData());
    }
}
