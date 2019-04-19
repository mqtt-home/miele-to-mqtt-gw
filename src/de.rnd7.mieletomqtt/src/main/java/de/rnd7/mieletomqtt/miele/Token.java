package de.rnd7.mieletomqtt.miele;

import org.json.JSONObject;

public class Token {
	
	private String accessToken;
	private String refreshToken;
	private int expiresIn;
	
	public static Token from(JSONObject answer) {
		Token result = new Token();
		
		result.accessToken = answer.getString("access_token");
		result.refreshToken = answer.getString("refresh_token");
		result.expiresIn = answer.getInt("expires_in");
		
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
	
	public int getExpiresIn() {
		return expiresIn;
	}
	
	public Token setExpiresIn(int expiresIn) {
		this.expiresIn = expiresIn;
		
		return this;
	}
	
}
