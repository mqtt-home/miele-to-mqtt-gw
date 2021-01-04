package de.rnd7.mieletomqtt;

import com.google.common.eventbus.EventBus;
import de.rnd7.miele.api.MieleDevice;
import de.rnd7.mqtt.Message;

import java.util.HashMap;
import java.util.Map;
import java.util.function.Consumer;

public class MieleEventHandler implements Consumer<MieleDevice> {
    private EventBus eventBus;
    private final boolean deduplicate;

    private final Map<String, String> messages = new HashMap<>();

    public MieleEventHandler(final EventBus eventBus, final boolean deduplicate) {
        this.eventBus = eventBus;
        this.deduplicate = deduplicate;
    }

    @Override
    public void accept(final MieleDevice mieleDevice) {
        final String message = mieleDevice.toFullMessage().toString();
        if (!handleDepduplication(mieleDevice, message)) {
            return;
        }

        this.eventBus.post(new Message(mieleDevice.getId() + "/full", message));
        this.eventBus.post(new Message(mieleDevice.getId(), mieleDevice.toSmallMessage().toString()));
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
