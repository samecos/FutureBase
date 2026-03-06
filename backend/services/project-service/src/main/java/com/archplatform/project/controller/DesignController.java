package com.archplatform.project.controller;

import com.archplatform.project.dto.*;
import com.archplatform.project.security.JwtAuthenticationFilter;
import com.archplatform.project.service.DesignService;
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
@RequestMapping("/projects/{projectId}/designs")
@RequiredArgsConstructor
public class DesignController {

    private final DesignService designService;

    @GetMapping
    public ResponseEntity<Page<DesignDTO>> getDesigns(@PathVariable UUID projectId,
                                                      Pageable pageable,
                                                      @AuthenticationPrincipal JwtAuthenticationFilter.UserPrincipal principal) {
        Page<DesignDTO> designs = designService.getDesignsByProject(projectId, principal.id(), pageable);
        return ResponseEntity.ok(designs);
    }

    @GetMapping("/{designId}")
    public ResponseEntity<DesignDTO> getDesign(@PathVariable UUID projectId,
                                                @PathVariable UUID designId,
                                                @AuthenticationPrincipal JwtAuthenticationFilter.UserPrincipal principal) {
        DesignDTO design = designService.getDesignById(designId, principal.id());
        return ResponseEntity.ok(design);
    }

    @PostMapping
    public ResponseEntity<DesignDTO> createDesign(@PathVariable UUID projectId,
                                                   @Valid @RequestBody CreateDesignRequest request,
                                                   @AuthenticationPrincipal JwtAuthenticationFilter.UserPrincipal principal) {
        DesignDTO design = designService.createDesign(projectId, principal.id(), request);
        return ResponseEntity.ok(design);
    }

    @PutMapping("/{designId}")
    public ResponseEntity<DesignDTO> updateDesign(@PathVariable UUID projectId,
                                                   @PathVariable UUID designId,
                                                   @Valid @RequestBody CreateDesignRequest request,
                                                   @AuthenticationPrincipal JwtAuthenticationFilter.UserPrincipal principal) {
        DesignDTO design = designService.updateDesign(designId, principal.id(), request);
        return ResponseEntity.ok(design);
    }

    @DeleteMapping("/{designId}")
    public ResponseEntity<Void> deleteDesign(@PathVariable UUID projectId,
                                              @PathVariable UUID designId,
                                              @AuthenticationPrincipal JwtAuthenticationFilter.UserPrincipal principal) {
        designService.deleteDesign(designId, principal.id());
        return ResponseEntity.ok().build();
    }

    @GetMapping("/search")
    public ResponseEntity<List<DesignDTO>> searchDesigns(@PathVariable UUID projectId,
                                                          @RequestParam String keyword,
                                                          @AuthenticationPrincipal JwtAuthenticationFilter.UserPrincipal principal) {
        List<DesignDTO> designs = designService.searchDesigns(projectId, keyword, principal.id());
        return ResponseEntity.ok(designs);
    }

    // Lock management
    @PostMapping("/{designId}/lock")
    public ResponseEntity<Boolean> acquireLock(@PathVariable UUID projectId,
                                                @PathVariable UUID designId,
                                                @RequestParam(defaultValue = "30") int durationMinutes,
                                                @AuthenticationPrincipal JwtAuthenticationFilter.UserPrincipal principal) {
        boolean locked = designService.acquireLock(designId, principal.id(), durationMinutes);
        return ResponseEntity.ok(locked);
    }

    @DeleteMapping("/{designId}/lock")
    public ResponseEntity<Void> releaseLock(@PathVariable UUID projectId,
                                             @PathVariable UUID designId,
                                             @AuthenticationPrincipal JwtAuthenticationFilter.UserPrincipal principal) {
        designService.releaseLock(designId, principal.id());
        return ResponseEntity.ok().build();
    }

    @PostMapping("/{designId}/move")
    public ResponseEntity<Void> moveDesign(@PathVariable UUID projectId,
                                            @PathVariable UUID designId,
                                            @RequestParam UUID targetFolderId,
                                            @AuthenticationPrincipal JwtAuthenticationFilter.UserPrincipal principal) {
        designService.moveDesign(designId, targetFolderId, principal.id());
        return ResponseEntity.ok().build();
    }
}
