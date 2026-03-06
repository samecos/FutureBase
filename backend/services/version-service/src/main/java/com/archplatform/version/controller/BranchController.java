package com.archplatform.version.controller;

import com.archplatform.version.dto.BranchDTO;
import com.archplatform.version.dto.CreateBranchRequest;
import com.archplatform.version.security.JwtAuthenticationFilter;
import com.archplatform.version.service.BranchService;
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
@RequestMapping("/branches")
@RequiredArgsConstructor
public class BranchController {

    private final BranchService branchService;

    @GetMapping("/{branchId}")
    public ResponseEntity<BranchDTO> getBranch(@PathVariable UUID branchId) {
        BranchDTO branch = branchService.getBranch(branchId);
        return ResponseEntity.ok(branch);
    }

    @GetMapping
    public ResponseEntity<List<BranchDTO>> getBranchesByDesign(@RequestParam UUID designId) {
        List<BranchDTO> branches = branchService.getBranchesByDesign(designId);
        return ResponseEntity.ok(branches);
    }

    @GetMapping("/default")
    public ResponseEntity<BranchDTO> getDefaultBranch(@RequestParam UUID designId) {
        BranchDTO branch = branchService.getDefaultBranch(designId);
        return ResponseEntity.ok(branch);
    }

    @PostMapping
    public ResponseEntity<BranchDTO> createBranch(
            @RequestParam UUID tenantId,
            @Valid @RequestBody CreateBranchRequest request,
            @AuthenticationPrincipal JwtAuthenticationFilter.UserPrincipal principal) {
        BranchDTO branch = branchService.createBranch(tenantId, principal.id(), request);
        return ResponseEntity.ok(branch);
    }

    @PutMapping("/{branchId}")
    public ResponseEntity<BranchDTO> updateBranch(
            @PathVariable UUID branchId,
            @RequestParam(required = false) String name,
            @RequestParam(required = false) String description,
            @AuthenticationPrincipal JwtAuthenticationFilter.UserPrincipal principal) {
        BranchDTO branch = branchService.updateBranch(branchId, principal.id(), name, description);
        return ResponseEntity.ok(branch);
    }

    @DeleteMapping("/{branchId}")
    public ResponseEntity<Void> deleteBranch(
            @PathVariable UUID branchId,
            @AuthenticationPrincipal JwtAuthenticationFilter.UserPrincipal principal) {
        branchService.deleteBranch(branchId, principal.id());
        return ResponseEntity.ok().build();
    }

    @PostMapping("/{branchId}/set-default")
    public ResponseEntity<Void> setDefaultBranch(
            @PathVariable UUID branchId,
            @RequestParam UUID designId) {
        branchService.setDefaultBranch(designId, branchId);
        return ResponseEntity.ok().build();
    }
}
