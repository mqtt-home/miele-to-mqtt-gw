package de.rnd7.miele;

import java.time.LocalDateTime;
import java.time.ZonedDateTime;

public class ConfigMieleToken {
    private String access;
    private String refresh;
    private ZonedDateTime validUntil;

    public String getAccess() {
        return access;
    }

    public void setAccess(final String access) {
        this.access = access;
    }

    public String getRefresh() {
        return refresh;
    }

    public void setRefresh(final String refresh) {
        this.refresh = refresh;
    }

    public ZonedDateTime getValidUntil() {
        return validUntil;
    }

    public void setValidUntil(final ZonedDateTime validUntil) {
        this.validUntil = validUntil;
    }

    public boolean isValid() {
        return access != null && refresh != null && validUntil != null;
    }
}
