package de.rnd7.mieletomqtt;

import de.rnd7.miele.api.MieleAPIState;
import de.rnd7.miele.api.MieleDevice;
import de.rnd7.miele.api.MieleEventListener;
import de.rnd7.mqtt.Message;

import java.util.HashMap;
import java.util.Map;

public class MieleEventHandler implements MieleEventListener {
    private final boolean deduplicate;

    private final Map<String, String> messages = new HashMap<>();

    private MieleAPIState state = MieleAPIState.unknown;

    public MieleEventHandler(final boolean deduplicate) {
        this.deduplicate = deduplicate;
    }

    @Override
    public void accept(final MieleDevice mieleDevice) {
        final String message = mieleDevice.toFullMessage().toString();
        if (!handleDepduplication(mieleDevice, message)) {
            return;
        }

        Events.post(new Message(mieleDevice.getId() + "/full", message));
        Events.post(new Message(mieleDevice.getId(), mieleDevice.toSmallMessage().toString()));
    }

    @Override
    public void state(final MieleAPIState state) {
        if (this.state != state) {
            this.state = state;
            Events.post(new Message(MIELE_STATE, state.toString()));
        }
    }

    private boolean handleDepduplication(final MieleDevice mieleDevice, final String message) {
        if (deduplicate) {
            if (messages.containsKey(mieleDevice.getId())) {
                if (messages.get(mieleDevice.getId()).equals(message)) {
                    return false;
                }
            }
            messages.put(mieleDevice.getId(), message);
        }
        return true;
    }

}
