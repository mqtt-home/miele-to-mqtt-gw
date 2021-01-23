package de.rnd7.mieletomqtt;

import de.rnd7.miele.api.MieleAPIState;
import de.rnd7.miele.api.MieleDevice;
import de.rnd7.miele.api.MieleEventListener;
import de.rnd7.mqttgateway.Events;
import de.rnd7.mqttgateway.PublishMessage;

public class MieleEventHandler implements MieleEventListener {

    private MieleAPIState state = MieleAPIState.unknown;

    @Override
    public void accept(final MieleDevice mieleDevice) {
        final String message = mieleDevice.toFullMessage().toString();

        Events.post(PublishMessage.relative(mieleDevice.getId() + "/full", message));
        Events.post(PublishMessage.relative(mieleDevice.getId(), mieleDevice.toSmallMessage().toString()));
    }

    @Override
    public void state(final MieleAPIState state) {
        if (this.state != state) {
            this.state = state;
            Events.post(PublishMessage.relative(MIELE_STATE, state.toString()));
        }
    }

}
