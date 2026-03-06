package com.archplatform.version.controller;

import com.archplatform.version.dto.*;
import com.archplatform.version.security.JwtAuthenticationFilter;
import com.archplatform.version.service.VersionService;
import jakarta.validation.Valid;
import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;
import org.springframework.data.domain.Page;
import org.springframework.data.domain.Pageable;
import org.springframework.http.ResponseEntity;
import org.springframework.security.core.annotation.AuthenticationPrincipal;
import org.springframework.web.bind.annotation.*;

import java.util.List;
import java.util.UUID;

@Slf4j
@RestController
@RequestMapping("/versions")
@RequiredArgsConstructor
public class VersionController {

    private final VersionService versionService;

    @GetMapping("/{versionId}")
    public ResponseEntity<VersionDTO> getVersion(@PathVariable UUID versionId) {
        VersionDTO version = versionService.getVersion(versionId);
        return ResponseEntity.ok(version);
    }

    @GetMapping
    public ResponseEntity<Page<VersionDTO>> getVersionsByBranch(
            @RequestParam UUID branchId,
            Pageable pageable) {
        Page<VersionDTO> versions = versionService.getVersionsByBranch(branchId, pageable);
        return ResponseEntity.ok(versions);
    }

    @GetMapping("/all")
    public ResponseEntity<List<VersionDTO>> getVersionsByDesign(@RequestParam UUID designId) {
        List<VersionDTO> versions = versionService.getVersionsByDesign(designId);
        return ResponseEntity.ok(versions);
    }

    @GetMapping("/latest")
    public ResponseEntity<VersionDTO> getLatestVersion(@RequestParam UUID branchId) {
        VersionDTO version = versionService.getLatestVersion(branchId);
        return ResponseEntity.ok(version);
    }

    @PostMapping
    public ResponseEntity<VersionDTO> createVersion(
            @RequestParam UUID tenantId,
            @Valid @RequestBody CreateVersionRequest request,
            @AuthenticationPrincipal JwtAuthenticationFilter.UserPrincipal principal) {
        VersionDTO version = versionService.createVersion(tenantId, principal.id(), request);
        return ResponseEntity.ok(version);
    }

    @PostMapping("/{versionId}/commit")
    public ResponseEntity<VersionDTO> commitVersion(
            @PathVariable UUID versionId,
            @RequestParam(required = false) String description,
            @AuthenticationPrincipal JwtAuthenticationFilter.UserPrincipal principal) {
        VersionDTO version = versionService.commitVersion(versionId, principal.id(), description);
        return ResponseEntity.ok(version);
    }

    @DeleteMapping("/{versionId}")
    public ResponseEntity<Void> deleteVersion(
            @PathVariable UUID versionId,
            @AuthenticationPrincipal JwtAuthenticationFilter.UserPrincipal principal) {
        versionService.deleteVersion(versionId, principal.id());
        return ResponseEntity.ok().build();
    }

    @GetMapping("/{versionId}/changes")
    public ResponseEntity<List<ChangeSetDTO>> getVersionChanges(@PathVariable UUID versionId) {
        List<ChangeSetDTO> changes = versionService.getVersionChanges(versionId);
        return ResponseEntity.ok(changes);
    }

    @GetMapping("/{versionId}/history")
    public ResponseEntity<List<VersionDTO>> getVersionHistory(
            @RequestParam UUID designId,
            @PathVariable UUID versionId,
            @RequestParam(required = false, defaultValue = "50") Integer limit) {
        List<VersionDTO> history = versionService.getVersionHistory(designId, versionId, limit);
        return ResponseEntity.ok(history);
    }

    @PostMapping("/{versionId}/rollback")
    public ResponseEntity<VersionDTO> rollbackToVersion(
            @PathVariable UUID versionId,
            @AuthenticationPrincipal JwtAuthenticationFilter.UserPrincipal principal) {
        VersionDTO version = versionService.rollbackToVersion(versionId, principal.id());
        return ResponseEntity.ok(version);
    }
}
