package com.archplatform.project.controller;

import com.archplatform.project.dto.*;
import com.archplatform.project.security.JwtAuthenticationFilter;
import com.archplatform.project.service.ProjectService;
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
@RequestMapping("/projects")
@RequiredArgsConstructor
public class ProjectController {

    private final ProjectService projectService;

    @GetMapping("/{projectId}")
    public ResponseEntity<ProjectDTO> getProject(@PathVariable UUID projectId,
                                                  @AuthenticationPrincipal JwtAuthenticationFilter.UserPrincipal principal) {
        ProjectDTO project = projectService.getProjectById(projectId, principal.id());
        return ResponseEntity.ok(project);
    }

    @GetMapping
    public ResponseEntity<Page<ProjectDTO>> getProjects(@RequestParam UUID tenantId,
                                                         Pageable pageable,
                                                         @AuthenticationPrincipal JwtAuthenticationFilter.UserPrincipal principal) {
        Page<ProjectDTO> projects = projectService.getProjectsByTenant(tenantId, principal.id(), pageable);
        return ResponseEntity.ok(projects);
    }

    @GetMapping("/my")
    public ResponseEntity<List<ProjectDTO>> getMyProjects(@AuthenticationPrincipal JwtAuthenticationFilter.UserPrincipal principal) {
        List<ProjectDTO> projects = projectService.getProjectsByUser(principal.id());
        return ResponseEntity.ok(projects);
    }

    @PostMapping
    public ResponseEntity<ProjectDTO> createProject(@RequestParam UUID tenantId,
                                                     @Valid @RequestBody CreateProjectRequest request,
                                                     @AuthenticationPrincipal JwtAuthenticationFilter.UserPrincipal principal) {
        ProjectDTO project = projectService.createProject(tenantId, principal.id(), request);
        return ResponseEntity.ok(project);
    }

    @PutMapping("/{projectId}")
    public ResponseEntity<ProjectDTO> updateProject(@PathVariable UUID projectId,
                                                     @Valid @RequestBody UpdateProjectRequest request,
                                                     @AuthenticationPrincipal JwtAuthenticationFilter.UserPrincipal principal) {
        ProjectDTO project = projectService.updateProject(projectId, principal.id(), request);
        return ResponseEntity.ok(project);
    }

    @DeleteMapping("/{projectId}")
    public ResponseEntity<Void> deleteProject(@PathVariable UUID projectId,
                                               @AuthenticationPrincipal JwtAuthenticationFilter.UserPrincipal principal) {
        projectService.deleteProject(projectId, principal.id());
        return ResponseEntity.ok().build();
    }

    @PostMapping("/{projectId}/archive")
    public ResponseEntity<Void> archiveProject(@PathVariable UUID projectId,
                                                @AuthenticationPrincipal JwtAuthenticationFilter.UserPrincipal principal) {
        projectService.archiveProject(projectId, principal.id());
        return ResponseEntity.ok().build();
    }

    // Members
    @GetMapping("/{projectId}/members")
    public ResponseEntity<List<ProjectMemberDTO>> getProjectMembers(@PathVariable UUID projectId,
                                                                     @AuthenticationPrincipal JwtAuthenticationFilter.UserPrincipal principal) {
        List<ProjectMemberDTO> members = projectService.getProjectMembers(projectId, principal.id());
        return ResponseEntity.ok(members);
    }

    @PostMapping("/{projectId}/members")
    public ResponseEntity<Void> addProjectMember(@PathVariable UUID projectId,
                                                  @Valid @RequestBody AddProjectMemberRequest request,
                                                  @AuthenticationPrincipal JwtAuthenticationFilter.UserPrincipal principal) {
        projectService.addProjectMember(projectId, principal.id(), request);
        return ResponseEntity.ok().build();
    }

    @DeleteMapping("/{projectId}/members/{memberId}")
    public ResponseEntity<Void> removeProjectMember(@PathVariable UUID projectId,
                                                     @PathVariable UUID memberId,
                                                     @AuthenticationPrincipal JwtAuthenticationFilter.UserPrincipal principal) {
        projectService.removeProjectMember(projectId, memberId, principal.id());
        return ResponseEntity.ok().build();
    }

    @PutMapping("/{projectId}/members/{memberId}/role")
    public ResponseEntity<Void> updateMemberRole(@PathVariable UUID projectId,
                                                  @PathVariable UUID memberId,
                                                  @RequestParam String role,
                                                  @AuthenticationPrincipal JwtAuthenticationFilter.UserPrincipal principal) {
        projectService.updateMemberRole(projectId, memberId, role, principal.id());
        return ResponseEntity.ok().build();
    }

    // Stats
    @GetMapping("/{projectId}/stats")
    public ResponseEntity<ProjectStatsDTO> getProjectStats(@PathVariable UUID projectId,
                                                            @AuthenticationPrincipal JwtAuthenticationFilter.UserPrincipal principal) {
        ProjectStatsDTO stats = projectService.getProjectStats(projectId, principal.id());
        return ResponseEntity.ok(stats);
    }
}
