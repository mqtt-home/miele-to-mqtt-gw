package de.rnd7.mieletomqtt.miele;

import com.google.common.eventbus.Subscribe;
import de.rnd7.mqtt.ReceivedMessage;

import java.util.ArrayList;
import java.util.List;

public class MessageListener {
    private List<ReceivedMessage> messages = new ArrayList<>();

    @Subscribe
    public void onMessage(ReceivedMessage message) {
        messages.add(message);
    }

    public List<ReceivedMessage> getMessages() {
        return messages;
    }
}
