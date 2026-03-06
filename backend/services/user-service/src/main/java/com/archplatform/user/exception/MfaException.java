package com.archplatform.user.exception;

public class MfaException extends RuntimeException {
    public MfaException(String message) {
        super(message);
    }
}
