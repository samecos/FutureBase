package com.archplatform.version.controller;

import com.archplatform.version.dto.CreateMergeRequest;
import com.archplatform.version.dto.MergeRequestDTO;
import com.archplatform.version.dto.VersionDiffDTO;
import com.archplatform.version.security.JwtAuthenticationFilter;
import com.archplatform.version.service.MergeService;
import jakarta.validation.Valid;
import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;
import org.springframework.http.ResponseEntity;
import org.springframework.security.core.annotation.AuthenticationPrincipal;
import org.springframework.web.bind.annotation.*;

import java.util.List;
import java.util.UUID;

@Slf4j
@RestController
@RequestMapping("/merges")
@RequiredArgsConstructor
public class MergeController {

    private final MergeService mergeService;

    @GetMapping("/{mergeRequestId}")
    public ResponseEntity<MergeRequestDTO> getMergeRequest(@PathVariable UUID mergeRequestId) {
        MergeRequestDTO mr = mergeService.getMergeRequest(mergeRequestId);
        return ResponseEntity.ok(mr);
    }

    @GetMapping
    public ResponseEntity<List<MergeRequestDTO>> getMergeRequestsByDesign(@RequestParam UUID designId) {
        List<MergeRequestDTO> requests = mergeService.getMergeRequestsByDesign(designId);
        return ResponseEntity.ok(requests);
    }

    @GetMapping("/open")
    public ResponseEntity<List<MergeRequestDTO>> getOpenMergeRequests(@RequestParam UUID designId) {
        List<MergeRequestDTO> requests = mergeService.getOpenMergeRequests(designId);
        return ResponseEntity.ok(requests);
    }

    @PostMapping
    public ResponseEntity<MergeRequestDTO> createMergeRequest(
            @RequestParam UUID tenantId,
            @Valid @RequestBody CreateMergeRequest request,
            @AuthenticationPrincipal JwtAuthenticationFilter.UserPrincipal principal) {
        MergeRequestDTO mr = mergeService.createMergeRequest(tenantId, principal.id(), request);
        return ResponseEntity.ok(mr);
    }

    @PostMapping("/{mergeRequestId}/merge")
    public ResponseEntity<MergeRequestDTO> performMerge(
            @PathVariable UUID mergeRequestId,
            @AuthenticationPrincipal JwtAuthenticationFilter.UserPrincipal principal) {
        MergeRequestDTO mr = mergeService.performMerge(mergeRequestId, principal.id());
        return ResponseEntity.ok(mr);
    }

    @PostMapping("/{mergeRequestId}/close")
    public ResponseEntity<Void> closeMergeRequest(
            @PathVariable UUID mergeRequestId,
            @AuthenticationPrincipal JwtAuthenticationFilter.UserPrincipal principal) {
        mergeService.closeMergeRequest(mergeRequestId, principal.id());
        return ResponseEntity.ok().build();
    }

    @GetMapping("/preview")
    public ResponseEntity<VersionDiffDTO> previewMerge(
            @RequestParam UUID sourceBranchId,
            @RequestParam UUID targetBranchId) {
        VersionDiffDTO diff = mergeService.previewMerge(sourceBranchId, targetBranchId);
        return ResponseEntity.ok(diff);
    }

    @PostMapping("/{mergeRequestId}/resolve")
    public ResponseEntity<MergeRequestDTO> resolveConflicts(
            @PathVariable UUID mergeRequestId,
            @RequestBody String resolution,
            @AuthenticationPrincipal JwtAuthenticationFilter.UserPrincipal principal) {
        MergeRequestDTO mr = mergeService.resolveConflicts(mergeRequestId, resolution, principal.id());
        return ResponseEntity.ok(mr);
    }
}
