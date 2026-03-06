package com.archplatform.user.security;

import com.archplatform.user.entity.Permission;
import com.archplatform.user.entity.Role;
import com.archplatform.user.entity.User;
import com.archplatform.user.repository.UserRepository;
import lombok.RequiredArgsConstructor;
import org.springframework.security.core.authority.SimpleGrantedAuthority;
import org.springframework.security.core.userdetails.UserDetails;
import org.springframework.security.core.userdetails.UserDetailsService;
import org.springframework.security.core.userdetails.UsernameNotFoundException;
import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Transactional;

import java.util.Collection;
import java.util.stream.Collectors;

@Service
@RequiredArgsConstructor
public class CustomUserDetailsService implements UserDetailsService {

    private final UserRepository userRepository;

    @Override
    @Transactional(readOnly = true)
    public UserDetails loadUserByUsername(String emailOrUsername) throws UsernameNotFoundException {
        User user = userRepository.findActiveByEmail(emailOrUsername)
            .orElseGet(() -> userRepository.findByUsername(emailOrUsername)
                .orElseThrow(() -> new UsernameNotFoundException("User not found: " + emailOrUsername)));

        if (user.isLocked()) {
            throw new UsernameNotFoundException("Account is locked");
        }

        return org.springframework.security.core.userdetails.User.builder()
            .username(user.getEmail())
            .password(user.getPasswordHash())
            .authorities(getAuthorities(user))
            .accountLocked(user.isLocked())
            .disabled(!user.isActive())
            .build();
    }

    private Collection<SimpleGrantedAuthority> getAuthorities(User user) {
        return user.getRoles().stream()
            .flatMap(role -> {
                // Add role authority
                var roleAuth = new SimpleGrantedAuthority("ROLE_" + role.getName());
                // Add permission authorities
                var permAuths = role.getPermissions().stream()
                    .map(Permission::getPermissionString)
                    .map(SimpleGrantedAuthority::new);
                return java.util.stream.Stream.concat(
                    java.util.stream.Stream.of(roleAuth),
                    permAuths
                );
            })
            .collect(Collectors.toSet());
    }
}
