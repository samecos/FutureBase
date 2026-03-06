package com.archplatform.user.security;

import com.warrenstrange.googleauth.GoogleAuthenticator;
import com.warrenstrange.googleauth.GoogleAuthenticatorConfig;
import com.warrenstrange.googleauth.GoogleAuthenticatorKey;
import com.warrenstrange.googleauth.GoogleAuthenticatorQRGenerator;
import lombok.extern.slf4j.Slf4j;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.stereotype.Service;

import java.util.concurrent.TimeUnit;

@Slf4j
@Service
public class MfaService {

    @Value("${security.mfa.issuer:ArchPlatform}")
    private String issuer;

    private final GoogleAuthenticator gAuth;

    public MfaService() {
        GoogleAuthenticatorConfig config = new GoogleAuthenticatorConfig.GoogleAuthenticatorConfigBuilder()
            .setTimeStepSizeInMillis(TimeUnit.SECONDS.toMillis(30))
            .setWindowSize(3) // Allow 3 time steps (90 seconds) of tolerance
            .setNumberOfScratchCodes(10)
            .build();
        
        this.gAuth = new GoogleAuthenticator(config);
    }

    public String generateSecret() {
        GoogleAuthenticatorKey key = gAuth.createCredentials();
        return key.getKey();
    }

    public String getQrCodeUrl(String email, String secret) {
        return GoogleAuthenticatorQRGenerator.getOtpAuthURL(issuer, email, new GoogleAuthenticatorKey.Builder(secret).build());
    }

    public boolean verifyCode(String secret, String code) {
        try {
            int verificationCode = Integer.parseInt(code);
            return gAuth.authorize(secret, verificationCode);
        } catch (NumberFormatException e) {
            log.warn("Invalid MFA code format: {}", code);
            return false;
        }
    }

    public String[] generateBackupCodes() {
        GoogleAuthenticatorKey key = gAuth.createCredentials();
        return key.getScratchCodes()
            .stream()
            .map(String::valueOf)
            .toArray(String[]::new);
    }
}
