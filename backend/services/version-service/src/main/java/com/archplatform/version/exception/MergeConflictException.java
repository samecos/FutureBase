package com.archplatform.version.exception;

public class MergeConflictException extends RuntimeException {
    public MergeConflictException(String message) {
        super(message);
    }
}
