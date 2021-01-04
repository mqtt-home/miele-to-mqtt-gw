package de.rnd7.miele.api;

import org.json.JSONObject;

import java.time.LocalDateTime;

public class Token {
	
	private String accessToken;
	private String refreshToken;
	private LocalDateTime expiresAt;
	
	public static Token from(JSONObject answer) {
		Token result = new Token();
		
		result.accessToken = answer.getString("access_token");
		result.refreshToken = answer.getString("refresh_token");
		result.expiresAt = LocalDateTime.now()
				.plusSeconds(answer.getInt("expires_in"))
				.minusHours(1);

		return result;
	}
	
	public String getAccessToken() {
		return accessToken;
	}
	
	public Token setAccessToken(String accessToken) {
		this.accessToken = accessToken;
		
		return this;
	}
	
	public String getRefreshToken() {
		return refreshToken;
	}
	
	public Token setRefreshToken(String refreshToken) {
		this.refreshToken = refreshToken;
		
		return this;
	}

	public LocalDateTime getExpiresAt() {
		return expiresAt;
	}

	public Token setExpiresAt(final LocalDateTime expiresAt) {
		this.expiresAt = expiresAt;

		return this;
	}

	@Override
	public String toString() {
		return "Token{" +
				"accessToken='" + accessToken + '\'' +
				", refreshToken='" + refreshToken + '\'' +
				", expiresAt=" + expiresAt +
				'}';
	}
}
