package de.rnd7.mieletomqtt.miele;

import com.google.common.eventbus.Subscribe;
import de.rnd7.mqttgateway.Message;

import java.util.ArrayList;
import java.util.List;

public class MessageListener {
    private List<Message> messages = new ArrayList<>();

    @Subscribe
    public void onMessage(Message message) {
        messages.add(message);
    }

    public List<Message> getMessages() {
        return messages;
    }
}
