package com.archplatform.user.controller;

import com.archplatform.user.dto.*;
import com.archplatform.user.security.JwtAuthenticationFilter;
import com.archplatform.user.service.UserService;
import jakarta.validation.Valid;
import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;
import org.springframework.data.domain.Page;
import org.springframework.data.domain.Pageable;
import org.springframework.http.ResponseEntity;
import org.springframework.security.access.prepost.PreAuthorize;
import org.springframework.security.core.annotation.AuthenticationPrincipal;
import org.springframework.web.bind.annotation.*;

import java.util.UUID;

@Slf4j
@RestController
@RequestMapping("/users")
@RequiredArgsConstructor
public class UserController {

    private final UserService userService;

    @GetMapping("/me")
    public ResponseEntity<UserDTO> getCurrentUser(@AuthenticationPrincipal JwtAuthenticationFilter.UserPrincipal principal) {
        UserDTO user = userService.getUserById(principal.id());
        return ResponseEntity.ok(user);
    }

    @PutMapping("/me")
    public ResponseEntity<UserDTO> updateCurrentUser(@AuthenticationPrincipal JwtAuthenticationFilter.UserPrincipal principal,
                                                      @Valid @RequestBody UpdateUserRequest request) {
        UserDTO user = userService.updateUser(principal.id(), request);
        return ResponseEntity.ok(user);
    }

    @PostMapping("/me/password")
    public ResponseEntity<Void> changePassword(@AuthenticationPrincipal JwtAuthenticationFilter.UserPrincipal principal,
                                                @Valid @RequestBody ChangePasswordRequest request) {
        userService.changePassword(principal.id(), request);
        return ResponseEntity.ok().build();
    }

    @DeleteMapping("/me")
    public ResponseEntity<Void> deleteCurrentUser(@AuthenticationPrincipal JwtAuthenticationFilter.UserPrincipal principal) {
        userService.deleteUser(principal.id());
        return ResponseEntity.ok().build();
    }

    // MFA endpoints
    @PostMapping("/me/mfa/setup")
    public ResponseEntity<MfaSetupResponse> setupMfa(@AuthenticationPrincipal JwtAuthenticationFilter.UserPrincipal principal) {
        MfaSetupResponse response = userService.setupMfa(principal.id());
        return ResponseEntity.ok(response);
    }

    @PostMapping("/me/mfa/verify")
    public ResponseEntity<Void> verifyAndEnableMfa(@AuthenticationPrincipal JwtAuthenticationFilter.UserPrincipal principal,
                                                    @RequestParam String code) {
        userService.verifyAndEnableMfa(principal.id(), code);
        return ResponseEntity.ok().build();
    }

    @PostMapping("/me/mfa/disable")
    public ResponseEntity<Void> disableMfa(@AuthenticationPrincipal JwtAuthenticationFilter.UserPrincipal principal,
                                            @RequestParam String password) {
        userService.disableMfa(principal.id(), password);
        return ResponseEntity.ok().build();
    }

    // Admin endpoints
    @GetMapping("/{userId}")
    @PreAuthorize("hasAuthority('user:read')")
    public ResponseEntity<UserDTO> getUserById(@PathVariable UUID userId) {
        UserDTO user = userService.getUserById(userId);
        return ResponseEntity.ok(user);
    }

    @GetMapping
    @PreAuthorize("hasAuthority('user:manage')")
    public ResponseEntity<Page<UserDTO>> getAllUsers(@RequestParam UUID tenantId, Pageable pageable) {
        Page<UserDTO> users = userService.getUsersByTenant(tenantId, pageable);
        return ResponseEntity.ok(users);
    }
}
