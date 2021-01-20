package de.rnd7.mieletomqtt.miele;

import com.google.common.eventbus.Subscribe;
import de.rnd7.mqtt.ReceivedMessage;

import java.util.Arrays;
import java.util.List;

public class EndToEndMessages extends MessageListener {

    private ReceivedMessage full;
    private ReceivedMessage small;
    private ReceivedMessage state;
    private ReceivedMessage miele;

    @Subscribe
    public void onMessage(ReceivedMessage message) {
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
    public List<ReceivedMessage> getMessages() {
        return Arrays.asList(full, small, state, miele);
    }

    private boolean isMiele(ReceivedMessage message) {
        return message.getTopic().equals("miele/bridge/miele");
    }

    private boolean isState(ReceivedMessage message) {
        return message.getTopic().equals("miele/bridge/state");
    }

    private boolean isFull(ReceivedMessage message) {
        return message.getTopic().endsWith("/full");
    }

    public ReceivedMessage getMiele() {
        return miele;
    }

    public ReceivedMessage getFull() {
        return full;
    }

    public ReceivedMessage getSmall() {
        return small;
    }

    public ReceivedMessage getState() {
        return state;
    }

    public boolean isFulfilled() {
        return  miele != null
            && state != null
            && small != null
            && full != null;
    }
}
