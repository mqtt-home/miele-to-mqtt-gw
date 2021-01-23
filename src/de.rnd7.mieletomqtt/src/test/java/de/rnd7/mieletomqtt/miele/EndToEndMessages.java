package de.rnd7.mieletomqtt.miele;

import com.google.common.eventbus.Subscribe;
import de.rnd7.mqttgateway.Message;

import java.util.Arrays;
import java.util.List;

public class EndToEndMessages extends MessageListener {

    private Message full;
    private Message small;
    private Message state;
    private Message miele;

    @Subscribe
    public void onMessage(Message message) {
        if (isMiele(message)) {
            miele = message;
        } else if (isState(message)) {
            state = message;
        } else if (isFull(message)) {
            full = message;
        } else {
            small = message;
        }
    }

    @Override
    public List<Message> getMessages() {
        return Arrays.asList(full, small, state, miele);
    }

    private boolean isMiele(Message message) {
        return message.getTopic().equals("miele/bridge/miele");
    }

    private boolean isState(Message message) {
        return message.getTopic().equals("miele/bridge/state");
    }

    private boolean isFull(Message message) {
        return message.getTopic().endsWith("/full");
    }

    public Message getMiele() {
        return miele;
    }

    public Message getFull() {
        return full;
    }

    public Message getSmall() {
        return small;
    }

    public Message getState() {
        return state;
    }

    public boolean isFulfilled() {
        return  miele != null
            && state != null
            && small != null
            && full != null;
    }
}
